package redis_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/pivotal-cf/brokerapi"
	"github.com/pivotal-cf/cf-redis-broker/brokerconfig"
	"github.com/pivotal-cf/cf-redis-broker/redis"
	"github.com/pivotal-cf/cf-redis-broker/redis/fakes"
	"github.com/pivotal-golang/lager/lagertest"

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
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("remote-repo")
		config = brokerconfig.Config{}
		config.RedisConfiguration.Dedicated.Nodes = []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"}
		config.RedisConfiguration.Dedicated.Port = 6379
		config.AgentPort = "1234"

		var err error
		tmpDir, err = ioutil.TempDir("", "cf-redis-broker")
		Expect(err).ToNot(HaveOccurred())

		fakeAgentClient = &fakes.FakeAgentClient{}
		fakeAgentClient.CredentialsFunc = func(rootURL string) (redis.Credentials, error) {
			return redis.Credentials{
				Port:     6666,
				Password: "password",
			}, nil
		}

		statefilePath = path.Join(tmpDir, "statefile.json")
		config.RedisConfiguration.Dedicated.StatefilePath = statefilePath
		repo, err = redis.NewRemoteRepository(fakeAgentClient, config, logger)
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("NewRemoteRepository", func() {
		Context("When a state file does not exist", func() {
			It("logs statefile creation", func() {
				Expect(logger).To(gbytes.Say(fmt.Sprintf("statefile %s not found, generating instead", statefilePath)))
			})

			It("logs 0 dedicated instances found", func() {
				Expect(logger).To(gbytes.Say("0 dedicated Redis instances found"))
			})
		})

		Context("When a state file exists", func() {
			var statefile Statefile

			BeforeEach(func() {
				statefile = Statefile{
					AvailableInstances: []*redis.Instance{
						&redis.Instance{Host: "10.0.0.1"},
						&redis.Instance{Host: "10.0.0.2"},
					},
					AllocatedInstances: []*redis.Instance{
						&redis.Instance{
							Host: "10.0.0.3",
							ID:   "dedicated-instance",
						},
					},
				}
				putStatefileContents(statefilePath, statefile)
			})

			Context("When the state file can be read", func() {
				It("loads its state from the state file", func() {
					repo, err := redis.NewRemoteRepository(fakeAgentClient, config, logger)
					Expect(err).ToNot(HaveOccurred())

					allocatedInstances, err := repo.AllInstances()
					Expect(err).ToNot(HaveOccurred())
					Expect(repo.InstanceLimit()).To(Equal(3))
					Expect(len(allocatedInstances)).To(Equal(1))
					Expect(*allocatedInstances[0]).To(Equal(*statefile.AllocatedInstances[0]))
				})

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

				It("logs that it is starting to look for dedicated instances, and in which file", func() {
					expectedOutput := fmt.Sprintf(
						"Starting dedicated instance lookup in statefile: %s",
						statefilePath,
					)
					Eventually(logger).Should(gbytes.Say(expectedOutput))
				})

				It("logs the instance count", func() {
					redis.NewRemoteRepository(fakeAgentClient, config, logger)
					Eventually(logger).Should(gbytes.Say("1 dedicated Redis instance found"))
				})

				It("logs the instance IDs", func() {
					redis.NewRemoteRepository(fakeAgentClient, config, logger)
					Eventually(logger).Should(gbytes.Say(
						fmt.Sprintf("Found dedicated instance: %s", statefile.AllocatedInstances[0].ID),
					))
				})
			})

			Context("When the state file cannot be read", func() {
				var err error

				BeforeEach(func() {
					os.Remove(statefilePath)
					os.Mkdir(statefilePath, 0644)
					_, err = redis.NewRemoteRepository(fakeAgentClient, config, logger)
				})

				AfterEach(func() {
					os.RemoveAll(statefilePath)
				})

				It("returns an error", func() {
					Expect(err).To(HaveOccurred())
				})

				It("logs the error", func() {
					Expect(logger).To(gbytes.Say("failed to read statefile"))
				})
			})

			Context("When the state file cannot be read due invalid JSON", func() {
				var newRepoErr error

				BeforeEach(func() {
					err := ioutil.WriteFile(statefilePath, []byte("NOT JSON"), 0644)
					Expect(err).ToNot(HaveOccurred())
					_, newRepoErr = redis.NewRemoteRepository(fakeAgentClient, config, logger)
				})

				It("returns an error", func() {
					Expect(newRepoErr).To(HaveOccurred())
				})

				It("logs the error", func() {
					Expect(logger).To(gbytes.Say("failed to read statefile due to invalid JSON"))
				})
			})
		})
	})

	Context("When no nodes are allocated", func() {

		Describe("#AllInstances", func() {
			var instances []*redis.Instance

			BeforeEach(func() {
				var err error

				instances, err = repo.AllInstances()
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns an empty array", func() {
				Expect(len(instances)).To(Equal(0))
			})
		})

		Describe("#AvailableNodes", func() {
			It("returns all the dedicated nodes", func() {
				instances := repo.AvailableInstances()
				Expect(instances[0].Host).To(Equal("10.0.0.1"))
				Expect(instances[1].Host).To(Equal("10.0.0.2"))
				Expect(instances[2].Host).To(Equal("10.0.0.3"))
			})
		})

		Describe("#InstanceCount", func() {
			It("returns the total number of allocated nodes", func() {
				Expect(repo.InstanceCount()).To(Equal(0))
			})
		})

		Describe("#Unbind", func() {
			It("returns an error", func() {
				err := repo.Unbind("NON-EXISTANT-INSTANCE", "SOME-BINDING")
				Expect(err).To(MatchError(brokerapi.ErrInstanceDoesNotExist))
			})
		})
	})

	Context("When one node is allocated", func() {
		BeforeEach(func() {
			err := repo.Create("foo")
			Expect(err).ToNot(HaveOccurred())
		})

		Describe("#Bind", func() {
			BeforeEach(func() {
				fakeAgentClient.CredentialsFunc = func(rootURL string) (redis.Credentials, error) {
					if rootURL == "https://10.0.0.1:1234" {
						return redis.Credentials{
							Port:     123456,
							Password: "super-secret",
						}, nil
					} else {
						return redis.Credentials{}, errors.New("wrong url")
					}
				}
			})

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
					os.Remove(statefilePath)
					os.Mkdir(statefilePath, 0644)
				})

				AfterEach(func() {
					os.RemoveAll(statefilePath)
				})

				It("does not bind", func() {
					_, err := repo.Bind("foo", "bar-binding")
					Expect(err).To(HaveOccurred())

					bindings, err := repo.BindingsForInstance("foo")
					Expect(err).ToNot(HaveOccurred())

					Expect(len(bindings)).To(Equal(0))
				})
			})
		})

		Describe("#Unbind", func() {
			Context("when the binding exists", func() {
				BeforeEach(func() {
					_, err := repo.Bind("foo", "foo-binding")
					Expect(err).ToNot(HaveOccurred())
				})

				It("returns successfully", func() {
					err := repo.Unbind("foo", "foo-binding")
					Expect(err).ToNot(HaveOccurred())
				})

				It("writes the new state to the statefile", func() {
					repo.Unbind("foo", "foo-binding")

					state := getStatefileContents(statefilePath)
					Expect(state.InstanceBindings["foo"]).To(BeEmpty())
				})

				Context("Concurrency", func() {
					It("prevents simultaneous edits", func() {
						chan1 := make(chan struct{})
						chan2 := make(chan struct{})

						action := func(control chan struct{}) {
							defer GinkgoRecover()
							repo.Unbind("foo", "foo-binding")
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
						os.Remove(statefilePath)
						os.Mkdir(statefilePath, 0644)
					})

					AfterEach(func() {
						os.RemoveAll(statefilePath)
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

		Describe("#AllInstances", func() {
			var instances []*redis.Instance

			BeforeEach(func() {
				var err error

				instances, err = repo.AllInstances()
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns an array with one IP address", func() {
				Expect(instances[0].Host).To(Equal("10.0.0.1"))
			})
		})

		Describe("#AvailableNodes", func() {
			It("returns all the dedicated nodes", func() {
				instances := repo.AvailableInstances()
				Expect(instances[0].Host).To(Equal("10.0.0.2"))
				Expect(instances[1].Host).To(Equal("10.0.0.3"))
			})
		})

		Describe("#Create", func() {
			It("allocates the next available node", func() {
				err := repo.Create("bar")
				Expect(err).ToNot(HaveOccurred())

				hosts := []string{}
				instances, err := repo.AllInstances()
				Expect(err).ToNot(HaveOccurred())

				for _, instance := range instances {
					hosts = append(hosts, instance.Host)
				}

				Expect(hosts).To(ContainElement("10.0.0.2"))
			})

			It("writes the new state to the statefile", func() {
				err := repo.Create("bar")
				Expect(err).ToNot(HaveOccurred())

				statefileContents := getStatefileContents(statefilePath)
				Expect(len(statefileContents.AllocatedInstances)).To(Equal(2))
				Expect(statefileContents.AllocatedInstances[1].Host).To(Equal("10.0.0.2"))
			})

			Context("when it cannot persist the state to the state file", func() {
				BeforeEach(func() {
					os.Remove(statefilePath)
					os.Mkdir(statefilePath, 0644)
				})

				AfterEach(func() {
					os.RemoveAll(statefilePath)
				})

				It("does not allocate an instance", func() {
					err := repo.Create("bar")
					Expect(err).To(HaveOccurred())

					_, err = repo.FindByID("bar")
					Expect(err).To(HaveOccurred())
				})
			})

			Context("when the instanceID is already allocated", func() {
				It("returns brokerapi.ErrInstanceAlreadyExists", func() {
					err := repo.Create("foo")
					Expect(err).To(HaveOccurred())
					Expect(err).To(Equal(brokerapi.ErrInstanceAlreadyExists))
				})
			})

			Context("when instance capacity has been reached", func() {
				BeforeEach(func() {
					repo.Create("bar")
					repo.Create("baz")
				})

				It("returns brokerapi.ErrInstanceLimitMet", func() {
					err := repo.Create("another")
					Expect(err).To(HaveOccurred())
					Expect(err).To(Equal(brokerapi.ErrInstanceLimitMet))
				})
			})
		})

		Describe("FindByID", func() {
			Context("when the instance exists", func() {
				It("returns the allocated instance", func() {
					instanceID := "foo"
					instance, err := repo.FindByID(instanceID)
					Expect(err).ToNot(HaveOccurred())
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
					Ω(result).Should(BeFalse())
					Ω(err).ShouldNot(HaveOccurred())
				})
			})

			Context("when instance exists", func() {
				It("returns true", func() {
					instanceID := "foo"
					result, err := repo.InstanceExists(instanceID)
					Ω(result).Should(BeTrue())
					Ω(err).ShouldNot(HaveOccurred())
				})
			})
		})

		Describe("#Destroy", func() {
			Context("when deleting an existing instance", func() {
				It("deallocates the instance", func() {
					err := repo.Destroy("foo")
					Expect(err).ToNot(HaveOccurred())
					Expect(len(repo.AvailableInstances())).To(Equal(3))
				})

				It("writes the new state to the statefile", func() {
					err := repo.Destroy("foo")
					Expect(err).ToNot(HaveOccurred())

					statefileContents := getStatefileContents(statefilePath)
					Expect(len(statefileContents.AllocatedInstances)).To(Equal(0))
				})

				It("resets the instance data", func() {
					instance, err := repo.FindByID("foo")
					Expect(err).ToNot(HaveOccurred())

					err = repo.Destroy("foo")
					Expect(err).ToNot(HaveOccurred())

					Expect(fakeAgentClient.ResetURLs).To(ConsistOf("https://" + instance.Host + ":1234"))
				})

				Context("when it cannot persist the state to the state file", func() {
					BeforeEach(func() {
						os.Remove(statefilePath)
						os.Mkdir(statefilePath, 0644)
					})

					AfterEach(func() {
						os.RemoveAll(statefilePath)
					})

					It("does not delete", func() {
						err := os.Chmod(tmpDir, 0444)
						Expect(err).ToNot(HaveOccurred())

						err = repo.Destroy("foo")
						Expect(err).To(HaveOccurred())

						_, err = repo.FindByID("foo")
						Expect(err).ToNot(HaveOccurred())
					})
				})

				Context("when the DELETE request fails", func() {
					clientError := errors.New("Internal server error")

					BeforeEach(func() {
						fakeAgentClient.ResetHandler = func(string) error {
							return clientError
						}
					})

					It("does not deallocate the instance", func() {
						repo.Destroy("foo")
						_, err := repo.FindByID("foo")
						Expect(err).ToNot(HaveOccurred())
					})

					It("does not modify the state file", func() {
						initialStatefileContents := getStatefileContents(statefilePath)
						repo.Destroy("foo")
						Expect(getStatefileContents(statefilePath)).To(Equal(initialStatefileContents))
					})

					It("returns the error", func() {
						err := repo.Destroy("foo")
						Expect(err).To(Equal(clientError))
					})
				})
			})

			Context("when deleting an instance that does not exist", func() {
				It("returns an error", func() {
					err := repo.Destroy("bar")
					Expect(err).To(HaveOccurred())
					Expect(err).To(Equal(brokerapi.ErrInstanceDoesNotExist))
				})
			})
		})
	})

	Context("When all nodes are allocated", func() {
		BeforeEach(func() {
			err := repo.Create("foo")
			Expect(err).ToNot(HaveOccurred())
			err = repo.Create("bar")
			Expect(err).ToNot(HaveOccurred())
			err = repo.Create("baz")
			Expect(err).ToNot(HaveOccurred())
		})

		Describe("#AllInstances", func() {
			var instances []*redis.Instance

			BeforeEach(func() {
				var err error

				instances, err = repo.AllInstances()
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns all nodes", func() {
				Expect(instances[0].Host).To(Equal("10.0.0.1"))
				Expect(instances[1].Host).To(Equal("10.0.0.2"))
				Expect(instances[2].Host).To(Equal("10.0.0.3"))
			})
		})

		Describe("#AvailableNodes", func() {
			It("returns an empty array", func() {
				instances := repo.AvailableInstances()
				Expect(len(instances)).To(Equal(0))
			})
		})

		Describe("#Create", func() {
			It("returns an error", func() {
				err := repo.Create("foo")
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(brokerapi.ErrInstanceLimitMet))
			})
		})

		Describe("#InstanceCount", func() {
			It("returns the total number of allocated nodes", func() {
				Expect(repo.InstanceCount()).To(Equal(3))
			})
		})
	})

	Describe("#PersistStatefile", func() {
		BeforeEach(func() {
			err := repo.Create("foo")
			Expect(err).ToNot(HaveOccurred())

			_, err = repo.Bind("foo", "foo-binding")
			Expect(err).ToNot(HaveOccurred())
		})

		It("writes state to a file", func() {
			err := repo.PersistStatefile()
			Expect(err).ToNot(HaveOccurred())

			statefileContents := getStatefileContents(statefilePath)

			Expect(len(statefileContents.AvailableInstances)).To(Equal(2))
			Expect(len(statefileContents.AllocatedInstances)).To(Equal(1))

			allocatedInstance := statefileContents.AllocatedInstances[0]
			Expect(allocatedInstance.Host).To(Equal("10.0.0.1"))

			Expect(len(statefileContents.InstanceBindings["foo"])).To(Equal(1))
			Expect(statefileContents.InstanceBindings["foo"][0]).To(Equal("foo-binding"))
		})
	})

	Describe("#IDForHost", func() {
		It("returns the corresponding instance ID", func() {
			err := repo.Create("foo")
			Expect(err).ToNot(HaveOccurred())

			Expect(repo.IDForHost(config.RedisConfiguration.Dedicated.Nodes[0])).To(Equal("foo"))
		})

		It("returns an empty string when the host is not allocated", func() {
			Expect(repo.IDForHost(config.RedisConfiguration.Dedicated.Nodes[0])).To(Equal(""))
		})

		It("returns an empty string when the host is unknown", func() {
			Expect(repo.IDForHost("nonsense")).To(Equal(""))
		})
	})
})
