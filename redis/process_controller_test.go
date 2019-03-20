package redis_test

import (
	"code.cloudfoundry.org/lager"
	"errors"
	"github.com/BooleanCat/igo/ios/iexec"
	"github.com/onsi/gomega/gbytes"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/lager/lagertest"
	"github.com/pivotal-cf/cf-redis-broker/redis"
	"github.com/pivotal-cf/cf-redis-broker/redis/fakes"
)

var _ = Describe("Redis Process Controller", func() {
	var (
		processController         *redis.OSProcessController
		instance                  *redis.Instance
		instanceInformer          *fakes.FakeInstanceInformer
		logger                    *lagertest.TestLogger
		processChecker            *fakes.FakeProcessChecker
		processKiller             *fakes.FakeProcessKiller
		pingServerFunc            redis.PingServerFunc
		waitUntilConnectableFunc  redis.WaitUntilConnectableFunc
		redisServerExecutablePath string
		connectionTimeoutError    error
		pingServerError           error
		exec                      *iexec.NestedCommandFake
		log                       *gbytes.Buffer
		err                       error
	)

	BeforeEach(func() {
		instance = new(redis.Instance)

		instanceInformer = new(fakes.FakeInstanceInformer)
		instanceInformer.InstancePidReturns(123, nil)

		processChecker = new(fakes.FakeProcessChecker)

		processKiller = new(fakes.FakeProcessKiller)
		processKiller.KillReturns(nil)

		exec = iexec.NewNestedCommandFake()

		logger = lagertest.NewTestLogger("process-controller")
		log = gbytes.NewBuffer()

		pingServerError = nil
		connectionTimeoutError = nil
		pingServerFunc = func(instance *redis.Instance) error { return pingServerError }
		waitUntilConnectableFunc = func(address *net.TCPAddr, timeout time.Duration) error { return connectionTimeoutError }
		redisServerExecutablePath = ""
	})

	JustBeforeEach(func() {
		processController = redis.NewOSProcessController(
			logger,
			instanceInformer,
			processChecker,
			processKiller,
			pingServerFunc,
			waitUntilConnectableFunc,
			redisServerExecutablePath,
		)
		processController.Exec = exec.Exec
		processController.Logger.RegisterSink(lager.NewWriterSink(log, lager.DEBUG))
	})

	itStartsARedisProcess := func(executablePath string) {
		command, args := exec.Exec.CommandArgsForCall(0)
		joinedArgs := strings.Join(args, " ")
		Expect(command).To(Equal(executablePath))
		Expect(joinedArgs).To(Equal("configFilePath --dir instanceDataDir --logfile logFilePath"))
	}

	Describe("StartAndWaitUntilReady", func() {
		It("runs the right command to start redis", func() {
			err = processController.StartAndWaitUntilReady(
				instance,
				"configFilePath",
				"instanceDataDir",
				"logFilePath",
				time.Second*1,
			)
			itStartsARedisProcess("redis-server")
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when the redis process fails to start", func() {
			BeforeEach(func() {
				connectionTimeoutError = errors.New("oops")
			})

			It("returns the same error that the WaitUntilConnectableFunc returns", func() {
				err = processController.StartAndWaitUntilReady(
					instance,
					"configFilePath",
					"instanceDataDir",
					"logFilePath",
					time.Second*1,
				)
				Expect(err).To(MatchError(connectionTimeoutError))
			})
		})
	})

	Describe("StartAndWaitUntilReadyWithConfig", func() {
		Context("When using a custom redis-server executable", func() {
			It("runs the right command to start redis", func() {
				processController.RedisServerExecutablePath = "custom/path/to/redis"

				args := []string{
					"configFilePath",
					"--dir", "instanceDataDir",
					"--logfile", "logFilePath",
				}
				err = processController.StartAndWaitUntilReadyWithConfig(instance, args, time.Second*1)
				Expect(err).NotTo(HaveOccurred())
				itStartsARedisProcess("custom/path/to/redis")
			})
		})

		It("runs the right command to start redis", func() {
			args := []string{
				"configFilePath",
				"--dir", "instanceDataDir",
				"--logfile", "logFilePath",
			}
			err = processController.StartAndWaitUntilReadyWithConfig(instance, args, time.Second*1)
			Expect(err).NotTo(HaveOccurred())
			itStartsARedisProcess("redis-server")
		})

		It("returns no error", func() {
			err := processController.StartAndWaitUntilReadyWithConfig(instance, []string{}, time.Second*1)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when the redis process fails to start", func() {
			BeforeEach(func() {
				connectionTimeoutError = errors.New("oops")
			})

			It("returns the same error that the WaitUntilConnectableFunc returns", func() {
				err = processController.StartAndWaitUntilReadyWithConfig(instance, []string{}, time.Second*1)
				Expect(err).To(MatchError(connectionTimeoutError))
			})
		})
	})

	Describe("Kill", func() {
		It("kills the correct process", func() {
			err = processController.Kill(instance)
			Expect(err).NotTo(HaveOccurred())

			Expect(processKiller.KillCallCount()).To(Equal(1))
			Expect(processKiller.KillArgsForCall(0)).To(Equal(123))
		})

		Context("when the pidfile does not exist", func() {
			BeforeEach(func() {
				instanceInformer.InstancePidReturns(0, errors.New("pid not found error"))
			})

			It("returns an error informing the operator to manually kill the redis process", func() {
				err = processController.Kill(instance)
				Expect(err).To(HaveOccurred())
				Eventually(log).Should(gbytes.Say("redis instance has no pidfile"))

				Expect(processKiller.KillCallCount()).To(Equal(0))
			})
		})
	})

	Describe("EnsureRunning", func() {
		Context("when the process is already running", func() {

			BeforeEach(func() {
				processChecker.AliveReturns(true)
			})

			It("runs and logs success", func() {
				err = processController.EnsureRunning(instance, "", "", "", "")
				Expect(err).NotTo(HaveOccurred())
				Eventually(log).Should(gbytes.Say("redis instance already running"))
			})

			Context("when the process is not the correct redis instance", func() {
				var file *os.File

				BeforeEach(func() {
					file, err = ioutil.TempFile("", "brokerTest")
					Expect(err).NotTo(HaveOccurred())

					pingServerError = errors.New("ping error")
				})

				It("restarts the redis-server", func() {
					By("running succesfully", func() {
						err = processController.EnsureRunning(instance, "/my/lovely/config", "", file.Name(), "")
						Expect(err).NotTo(HaveOccurred())
					})

					By("logging the failed PING", func() {
						Eventually(log).Should(gbytes.Say("failed to PING redis-server"))
					})

					By("deleting the pidfile", func() {
						_, err = os.Stat(file.Name())
						Expect(err).To(HaveOccurred())
					})

					By("logging the pidfile deletion", func() {
						Eventually(log).Should(gbytes.Say("removed stale pidfile"))
					})

					By("restarting the redis-server", func() {
						ran := false
						callCount := exec.Exec.CommandCallCount()
						for i := 0; i < callCount; i++ {
							command, args := exec.Exec.CommandArgsForCall(i)
							joinedArgs := strings.Join(args, " ")
							if command == "redis-server" && strings.Contains(joinedArgs, "/my/lovely/config") {
								ran = true
								break
							}
						}
						Expect(ran).To(BeTrue())
					})
				})

				Context("when the pidfile cannot be deleted", func() {
					It("logs the failure and returns an error", func() {
						err = processController.EnsureRunning(instance, "", "", "/not/a/valid/pidfile", "")
						Expect(err).To(HaveOccurred())
						Eventually(log).Should(gbytes.Say("failed to delete stale pidfile"))
					})
				})
			})
		})

		Context("when the process is not already running", func() {
			BeforeEach(func() {
				processChecker.AliveReturns(false)
			})

			It("starts the redis server", func() {
				err = processController.EnsureRunning(instance, "configFilePath", "instanceDataDir", "pidfilePath", "logFilePath")
				Expect(err).NotTo(HaveOccurred())
				itStartsARedisProcess("redis-server")
			})

			Context("when the redis server cannot be started", func() {
				BeforeEach(func() {
					exec.Cmd.RunReturns(errors.New("run error"))
				})

				It("should return an error", func() {
					err = processController.EnsureRunning(instance, "", "", "", "")
					Expect(err).To(HaveOccurred())
				})
			})
		})
	})
})
