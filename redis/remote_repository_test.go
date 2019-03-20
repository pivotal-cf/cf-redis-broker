package redis_test

import (
	"code.cloudfoundry.org/lager"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"code.cloudfoundry.org/lager/lagertest"
	"github.com/pivotal-cf/brokerapi"
	"github.com/pivotal-cf/cf-redis-broker/brokerconfig"
	"github.com/pivotal-cf/cf-redis-broker/redis"
	"github.com/pivotal-cf/cf-redis-broker/redis/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("RemoteRepository", func() {
	var (
		repo            *redis.RemoteRepository
		statefilePath   string
		tmpDir          string
		config          brokerconfig.Config
		fakeAgentClient *fakes.FakeAgentClient
		logger          *lagertest.TestLogger
		err             error
		log             *gbytes.Buffer
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("remote-repo")
		log = gbytes.NewBuffer()
		logger.RegisterSink(lager.NewWriterSink(log, lager.DEBUG))

		config = brokerconfig.Config{}
		config.RedisConfiguration.Dedicated.Nodes = []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"}
		config.RedisConfiguration.Dedicated.Port = 6379
		config.AgentPort = "1234"

		tmpDir, err = ioutil.TempDir("", "cf-redis-broker")
		Expect(err).NotTo(HaveOccurred())

		fakeAgentClient = new(fakes.FakeAgentClient)
		fakeAgentClient.CredentialsReturns(redis.Credentials{Port: 6666, Password: "password"}, nil)

		statefilePath = path.Join(tmpDir, "statefile.json")
		config.RedisConfiguration.Dedicated.StatefilePath = statefilePath
	})

	AfterEach(func() {
		err = os.RemoveAll(tmpDir)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("NewRemoteRepository", func() {
		Context("When a state file exists", func() {
			var statefile Statefile

			BeforeEach(func() {
				statefile = Statefile{
					AvailableInstances: []*redis.Instance{
						{Host: "10.0.0.1"},
						{Host: "10.0.0.2"},
					},
					AllocatedInstances: []*redis.Instance{
						{
							Host: "10.0.0.3",
							ID:   "dedicated-instance",
						},
					},
				}
				putStatefileContents(statefilePath, statefile)
			})

			Context("When the state file can be read", func() {
				It("creates a new remote repository", func() {
					repo, err := redis.NewRemoteRepository(fakeAgentClient, config, logger)
					Expect(err).ToNot(HaveOccurred())

					By("loading its state from the state file", func() {
						allocatedInstances, err := repo.AllInstances()
						Expect(err).ToNot(HaveOccurred())
						Expect(repo.InstanceLimit()).To(Equal(3))
						Expect(len(allocatedInstances)).To(Equal(1))
						Expect(*allocatedInstances[0]).To(Equal(*statefile.AllocatedInstances[0]))
					})

					By("logging that it is starting to look for dedicated instances, and in which file", func() {
						expectedOutput := fmt.Sprintf(
							"Starting dedicated instance lookup in statefile: %s",
							statefilePath,
						)
						Eventually(log).Should(gbytes.Say(expectedOutput))
					})

					By("logging the instance count", func() {
						Eventually(log).Should(gbytes.Say("1 dedicated Redis instance found"))
					})

					By("logging the instance IDs", func() {
						Eventually(log).Should(gbytes.Say(
							fmt.Sprintf("Found dedicated instance: %s", statefile.AllocatedInstances[0].ID),
						))
					})
				})

				Context("when there are changes to the config", func() {
					It("adds new nodes from config", func() {
						nodes := append(config.RedisConfiguration.Dedicated.Nodes, "10.0.0.4")
						config.RedisConfiguration.Dedicated.Nodes = nodes

						repo, err := redis.NewRemoteRepository(fakeAgentClient, config, logger)
						Expect(err).ToNot(HaveOccurred())

						availableInstances := repo.AvailableInstances()
						Expect(len(availableInstances)).To(Equal(3))
						Expect(availableInstances[2].Host).To(Equal("10.0.0.4"))
					})

					It("saves the statefile", func() {
						nodes := append(config.RedisConfiguration.Dedicated.Nodes, "10.0.0.4")
						config.RedisConfiguration.Dedicated.Nodes = nodes

						_, err := redis.NewRemoteRepository(fakeAgentClient, config, logger)
						Expect(err).ToNot(HaveOccurred())

						state := getStatefileContents(statefilePath)
						Expect(len(state.AvailableInstances)).To(Equal(3))
						Expect(state.AvailableInstances[2].Host).To(Equal("10.0.0.4"))
					})
				})
			})

			Context("When the state file has invalid permissions", func() {
				BeforeEach(func() {
					err = os.Remove(statefilePath)
					Expect(err).NotTo(HaveOccurred())
					err = os.Mkdir(statefilePath, 0644)
					Expect(err).NotTo(HaveOccurred())
				})

				AfterEach(func() {
					err = os.RemoveAll(statefilePath)
					Expect(err).NotTo(HaveOccurred())
				})

				It("logs the failure and returns an error", func() {
					_, err = redis.NewRemoteRepository(fakeAgentClient, config, logger)
					Expect(err).To(HaveOccurred())
					Eventually(log).Should(gbytes.Say("failed to read statefile"))
				})
			})

			Context("When the state file cannot be read due invalid JSON", func() {
				var newRepoErr error

				BeforeEach(func() {
					err := ioutil.WriteFile(statefilePath, []byte("NOT JSON"), 0644)
					Expect(err).NotTo(HaveOccurred())
				})

				It("logs the failure and returns an error", func() {
					_, newRepoErr = redis.NewRemoteRepository(fakeAgentClient, config, logger)
					Expect(newRepoErr).To(HaveOccurred())
					Eventually(log).Should(gbytes.Say("failed to read statefile due to invalid JSON"))
				})
			})
		})

		Context("When a state file does not exist", func() {
			It("logs statefile creation", func() {
				_, err = redis.NewRemoteRepository(fakeAgentClient, config, logger)
				Expect(err).NotTo(HaveOccurred())
				Expect(logger).To(gbytes.Say(fmt.Sprintf("statefile %s not found, generating instead", statefilePath)))
			})

			It("logs 0 dedicated instances found", func() {
				_, err = redis.NewRemoteRepository(fakeAgentClient, config, logger)
				Expect(err).NotTo(HaveOccurred())
				Expect(logger).To(gbytes.Say("0 dedicated Redis instances found"))
			})
		})
	})

	Context("When no nodes are allocated", func() {
		BeforeEach(func() {
			repo, err = redis.NewRemoteRepository(fakeAgentClient, config, logger)
			Expect(err).NotTo(HaveOccurred())
		})

		Describe("AllInstances", func() {
			var instances []*redis.Instance

			It("returns an empty array", func() {
				instances, err = repo.AllInstances()
				Expect(err).ToNot(HaveOccurred())
				Expect(instances).To(HaveLen(0))
			})
		})

		Describe("AvailableNodes", func() {
			It("returns all the dedicated nodes", func() {
				instances := repo.AvailableInstances()
				Expect(instances[0].Host).To(Equal("10.0.0.1"))
				Expect(instances[1].Host).To(Equal("10.0.0.2"))
				Expect(instances[2].Host).To(Equal("10.0.0.3"))
			})
		})

		Describe("InstanceCount", func() {
			It("returns the total number of allocated nodes", func() {
				Expect(repo.InstanceCount()).To(Equal(0))
			})
		})

		Describe("Unbind", func() {
			It("returns an error", func() {
				err := repo.Unbind("NON-EXISTANT-INSTANCE", "SOME-BINDING")
				Expect(err).To(MatchError(brokerapi.ErrInstanceDoesNotExist))
			})
		})
	})

	Context("When one node is allocated", func() {
		BeforeEach(func() {
			repo, err = redis.NewRemoteRepository(fakeAgentClient, config, logger)
			Expect(err).NotTo(HaveOccurred())

			err = repo.Create("foo")
			Expect(err).NotTo(HaveOccurred())

			fakeAgentClient.CredentialsStub = func(host string) (redis.Credentials, error) {
				if host == "10.0.0.1" {
					return redis.Credentials{
						Port:     123456,
						Password: "super-secret",
					}, nil
				} else {
					return redis.Credentials{}, errors.New("wrong url")
				}
			}
		})

		Describe("Bind", func() {
			It("returns the instance information", func() {
				instanceCredentials, err := repo.Bind("foo", "foo-binding")
				Expect(err).ToNot(HaveOccurred())
				Expect(instanceCredentials.Host).To(Equal("10.0.0.1"))
				Expect(instanceCredentials.Port).To(Equal(123456))
				Expect(instanceCredentials.Password).To(Equal("super-secret"))
			})

			It("writes the new state to the statefile", func() {
				_, err := repo.Bind("foo", "foo-binding")
				Expect(err).ToNot(HaveOccurred())

				statefileContents := getStatefileContents(statefilePath)
				Expect(len(statefileContents.InstanceBindings["foo"])).To(Equal(1))
				Expect(statefileContents.InstanceBindings["foo"][0]).To(Equal("foo-binding"))
			})

			Context("when it cannot persist the state to the state file", func() {
				BeforeEach(func() {
					err = os.Remove(statefilePath)
					Expect(err).NotTo(HaveOccurred())
					err = os.Mkdir(statefilePath, 0644)
					Expect(err).NotTo(HaveOccurred())
				})

				AfterEach(func() {
					err = os.RemoveAll(statefilePath)
					Expect(err).NotTo(HaveOccurred())
				})

				It("does not bind", func() {
					By("returning an error when binding", func() {
						_, err := repo.Bind("foo", "bar-binding")
						Expect(err).To(HaveOccurred())
					})

					By("reporting no bindings", func() {
						bindings, err := repo.BindingsForInstance("foo")
						Expect(err).ToNot(HaveOccurred())
						Expect(bindings).To(HaveLen(0))
					})
				})
			})
		})

		Describe("Unbind", func() {
			Context("when the binding exists", func() {
				BeforeEach(func() {
					_, err := repo.Bind("foo", "foo-binding")
					Expect(err).ToNot(HaveOccurred())
				})

				It("succesfully writes the new state to the statefile", func() {
					err = repo.Unbind("foo", "foo-binding")
					Expect(err).NotTo(HaveOccurred())

					state := getStatefileContents(statefilePath)
					Expect(state.InstanceBindings["foo"]).To(BeEmpty())
				})

				Context("when two concurrent unbind calls occur", func() {
					It("prevents simultaneous edits", func() {
						chan1 := make(chan struct{})
						chan2 := make(chan struct{})

						action := func(control chan struct{}) {
							defer GinkgoRecover()
							_ = repo.Unbind("foo", "foo-binding")
							close(control)
						}

						repo.Lock()

						go action(chan1)

						Consistently(chan1).ShouldNot(BeClosed())

						repo.Unlock()

						Eventually(chan1).Should(BeClosed())

						go action(chan2)

						Eventually(chan2).Should(BeClosed())
					})
				})

				Context("when the statefile cannot be persisted", func() {
					BeforeEach(func() {
						err = os.Remove(statefilePath)
						Expect(err).NotTo(HaveOccurred())
						err = os.Mkdir(statefilePath, 0644)
						Expect(err).NotTo(HaveOccurred())
					})

					AfterEach(func() {
						err = os.RemoveAll(statefilePath)
						Expect(err).NotTo(HaveOccurred())
					})

					It("does not unbind", func() {
						err := repo.Unbind("foo", "bar-binding")
						Expect(err).To(HaveOccurred())

						bindings, err := repo.BindingsForInstance("foo")
						Expect(err).ToNot(HaveOccurred())

						Expect(bindings).To(HaveLen(1))
					})
				})
			})

			Context("when the binding does not exist", func() {
				It("returns an error", func() {
					err := repo.Unbind("foo", "bar-binding")
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Describe("AllInstances", func() {
			var instances []*redis.Instance

			It("successfully returns an array with one IP address", func() {
				instances, err = repo.AllInstances()
				Expect(err).NotTo(HaveOccurred())
				Expect(instances[0].Host).To(Equal("10.0.0.1"))
			})
		})

		Describe("AvailableNodes", func() {
			It("returns all the dedicated nodes", func() {
				instances := repo.AvailableInstances()
				Expect(instances).To(HaveLen(2))
				Expect(instances[0].Host).To(Equal("10.0.0.2"))
				Expect(instances[1].Host).To(Equal("10.0.0.3"))
			})
		})

		Describe("Create", func() {
			It("allocates the next available node", func() {
				var hosts []string
				err = repo.Create("bar")
				Expect(err).ToNot(HaveOccurred())

				instances, err := repo.AllInstances()
				Expect(err).ToNot(HaveOccurred())
				Expect(instances).To(HaveLen(2))

				for _, instance := range instances {
					hosts = append(hosts, instance.Host)
				}

				Expect(hosts).To(ConsistOf("10.0.0.1", "10.0.0.2"))
			})

			It("writes the new state to the statefile", func() {
				err = repo.Create("bar")
				Expect(err).ToNot(HaveOccurred())

				statefileContents := getStatefileContents(statefilePath)
				Expect(len(statefileContents.AllocatedInstances)).To(Equal(2))
				Expect(statefileContents.AllocatedInstances[1].Host).To(Equal("10.0.0.2"))
			})

			It("logs that the instance was provisioned", func() {
				err = repo.Create("bar")
				Expect(err).ToNot(HaveOccurred())

				Eventually(log).Should(gbytes.Say("provision-instance"))
				Eventually(log).Should(gbytes.Say(`{"instance_id":"bar","message":"Successfully provisioned Redis instance","plan":"dedicated-vm"}`))
			})

			Context("when the state cannot be persisted to the state file", func() {
				BeforeEach(func() {
					err = os.Remove(statefilePath)
					Expect(err).NotTo(HaveOccurred())
					err = os.Mkdir(statefilePath, 0644)
					Expect(err).NotTo(HaveOccurred())
				})

				AfterEach(func() {
					err = os.RemoveAll(statefilePath)
					Expect(err).NotTo(HaveOccurred())
				})

				It("does not allocate an instance", func() {
					err = repo.Create("bar")
					Expect(err).To(HaveOccurred())

					_, err = repo.FindByID("bar")
					Expect(err).To(HaveOccurred())
				})
			})

			Context("when the instanceID is already allocated", func() {
				It("returns brokerapi.ErrInstanceAlreadyExists", func() {
					err := repo.Create("foo")
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError(brokerapi.ErrInstanceAlreadyExists))
				})
			})

			Context("when instance capacity has been reached", func() {
				BeforeEach(func() {
					err = repo.Create("bar")
					Expect(err).NotTo(HaveOccurred())
					err = repo.Create("baz")
					Expect(err).NotTo(HaveOccurred())
				})

				It("returns brokerapi.ErrInstanceLimitMet", func() {
					err = repo.Create("another")
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError(brokerapi.ErrInstanceLimitMet))
				})
			})
		})

		Describe("FindByID", func() {
			Context("when the instance exists", func() {
				It("returns the allocated instance", func() {
					instanceID := "foo"
					instance, err := repo.FindByID(instanceID)
					Expect(err).NotTo(HaveOccurred())
					Expect(instance.ID).To(Equal(instanceID))
					Expect(instance.Host).To(Equal("10.0.0.1"))
				})
			})

			Context("when the instance does not exist", func() {
				It("returns an error", func() {
					instanceID := "bar"
					_, err := repo.FindByID(instanceID)
					Expect(err).To(HaveOccurred())
					Expect(err).To(Equal(brokerapi.ErrInstanceDoesNotExist))
				})
			})
		})

		Describe("InstanceExists", func() {
			Context("when instance does not exist", func() {
				It("returns false", func() {
					instanceID := "bar"
					result, err := repo.InstanceExists(instanceID)
					Expect(result).To(BeFalse())
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Context("when instance exists", func() {
				It("returns true", func() {
					instanceID := "foo"
					result, err := repo.InstanceExists(instanceID)
					Expect(result).To(BeTrue())
					Expect(err).NotTo(HaveOccurred())
				})
			})
		})

		Describe("Destroy", func() {
			Context("when deleting an existing instance", func() {
				It("de-allocates the instance", func() {
					err = repo.Destroy("foo")
					Expect(err).NotTo(HaveOccurred())
					Expect(repo.AvailableInstances()).To(HaveLen(3))
				})

				It("writes the new state to the statefile", func() {
					err = repo.Destroy("foo")
					Expect(err).NotTo(HaveOccurred())

					statefileContents := getStatefileContents(statefilePath)
					Expect(statefileContents.AllocatedInstances).To(HaveLen(0))
				})

				It("resets the instance data", func() {
					instance, err := repo.FindByID("foo")
					Expect(err).NotTo(HaveOccurred())

					err = repo.Destroy("foo")
					Expect(err).NotTo(HaveOccurred())

					Expect(fakeAgentClient.ResetCallCount()).To(Equal(1))
					Expect(fakeAgentClient.ResetArgsForCall(0)).To(Equal(instance.Host))
				})

				It("logs that the instance was deprovisioned", func() {
					err = repo.Destroy("foo")
					Expect(err).NotTo(HaveOccurred())

					Eventually(log).Should(gbytes.Say("deprovision-instance"))
					Eventually(log).Should(gbytes.Say(`{"instance_id":"foo","message":"Successfully deprovisioned Redis instance","plan":"dedicated-vm"}`))
				})

				Context("when it cannot persist the state to the state file", func() {
					BeforeEach(func() {
						err = os.Remove(statefilePath)
						Expect(err).NotTo(HaveOccurred())

						err = os.Mkdir(statefilePath, 0644)
						Expect(err).NotTo(HaveOccurred())
					})

					It("does not delete the instance", func() {
						err = repo.Destroy("foo")
						Expect(err).To(HaveOccurred())

						_, err = repo.FindByID("foo")
						Expect(err).NotTo(HaveOccurred())
					})
				})

				Context("when the DELETE request fails", func() {
					var initialStatefileContents Statefile

					BeforeEach(func() {
						fakeAgentClient.ResetReturns(errors.New("internal server error"))
						initialStatefileContents = getStatefileContents(statefilePath)
					})

					It("does not deallocate the instance", func() {
						By("returning an error", func() {
							err = repo.Destroy("foo")
							Expect(err).To(MatchError("internal server error"))
						})

						By("not deleting the instance", func() {
							_, err = repo.FindByID("foo")
							Expect(err).NotTo(HaveOccurred())
						})

						By("not modifying the state file", func() {
							Expect(getStatefileContents(statefilePath)).To(Equal(initialStatefileContents))
						})
					})
				})
			})

			Context("when deleting an instance that does not exist", func() {
				It("returns an error", func() {
					err = repo.Destroy("bar")
					Expect(err).To(MatchError(brokerapi.ErrInstanceDoesNotExist))
				})
			})
		})
	})

	Context("When all nodes are allocated", func() {
		BeforeEach(func() {
			repo, err = redis.NewRemoteRepository(fakeAgentClient, config, logger)
			Expect(err).NotTo(HaveOccurred())

			err = repo.Create("foo")
			Expect(err).NotTo(HaveOccurred())
			err = repo.Create("bar")
			Expect(err).NotTo(HaveOccurred())
			err = repo.Create("baz")
			Expect(err).NotTo(HaveOccurred())
		})

		Describe("AllInstances", func() {
			It("returns all nodes", func() {
				instances, err := repo.AllInstances()
				Expect(err).NotTo(HaveOccurred())

				Expect(instances[0].Host).To(Equal("10.0.0.1"))
				Expect(instances[1].Host).To(Equal("10.0.0.2"))
				Expect(instances[2].Host).To(Equal("10.0.0.3"))
			})
		})

		Describe("AvailableNodes", func() {
			It("returns an empty array", func() {
				instances := repo.AvailableInstances()
				Expect(instances).To(HaveLen(0))
			})
		})

		Describe("Create", func() {
			It("returns an error", func() {
				err = repo.Create("foo")
				Expect(err).To(MatchError(brokerapi.ErrInstanceLimitMet))
			})
		})

		Describe("InstanceCount", func() {
			It("returns the total number of allocated nodes", func() {
				Expect(repo.InstanceCount()).To(Equal(3))
			})
		})
	})

	Describe("PersistStatefile", func() {
		BeforeEach(func() {
			repo, err = redis.NewRemoteRepository(fakeAgentClient, config, logger)
			Expect(err).NotTo(HaveOccurred())

			err = repo.Create("foo")
			Expect(err).NotTo(HaveOccurred())

			_, err = repo.Bind("foo", "foo-binding")
			Expect(err).NotTo(HaveOccurred())
		})

		It("writes state to a file", func() {
			err = repo.PersistStatefile()
			Expect(err).NotTo(HaveOccurred())

			statefileContents := getStatefileContents(statefilePath)

			Expect(statefileContents.AvailableInstances).To(HaveLen(2))
			Expect(statefileContents.AllocatedInstances).To(HaveLen(1))

			allocatedInstance := statefileContents.AllocatedInstances[0]
			Expect(allocatedInstance.Host).To(Equal("10.0.0.1"))

			Expect(statefileContents.InstanceBindings["foo"]).To(HaveLen(1))
			Expect(statefileContents.InstanceBindings["foo"][0]).To(Equal("foo-binding"))
		})

		It("sets the statefile permissions", func() {
			info, err := os.Stat(statefilePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(getPermissions(info)).To(Equal(0640))

		})
	})

	Describe("IDForHost", func() {
		BeforeEach(func() {
			repo, err = redis.NewRemoteRepository(fakeAgentClient, config, logger)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when there is a host allocated", func() {
			BeforeEach(func() {
				err = repo.Create("foo")
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns the corresponding instance ID", func() {
				Expect(repo.IDForHost(config.RedisConfiguration.Dedicated.Nodes[0])).To(Equal("foo"))
			})
		})

		Context("when there are no hosts allocated", func() {
			It("returns an empty string", func() {
				Expect(repo.IDForHost(config.RedisConfiguration.Dedicated.Nodes[0])).To(Equal(""))
			})
		})

		Context("when the host is unknown", func() {
			It("returns an empty string", func() {
				Expect(repo.IDForHost("nonsense")).To(Equal(""))
			})
		})
	})
})
