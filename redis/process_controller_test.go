package redis

import (
	"errors"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"time"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	"github.com/BooleanCat/igo/ios/iexec"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

type fakeProcessChecker struct {
	alive          bool
	lastCheckedPid int
}

func (fakeProcessChecker *fakeProcessChecker) Alive(pid int) bool {
	fakeProcessChecker.lastCheckedPid = pid
	return fakeProcessChecker.alive
}

type fakeProcessKiller struct {
	killed        bool
	lastPidKilled int
}

func (fakeProcessKiller *fakeProcessKiller) Kill(pid int) error {
	fakeProcessKiller.lastPidKilled = pid
	fakeProcessKiller.killed = true
	return nil
}

type fakeInstanceInformer struct{}

func (*fakeInstanceInformer) InstancePid(instanceID string) (int, error) {
	return 123, nil
}

var _ = Describe("Redis Process Controller", func() {
	var (
		processController    *OSProcessController
		instance             *Instance = new(Instance)
		instanceInformer     *fakeInstanceInformer
		logger               *lagertest.TestLogger
		fakeProcessChecker   *fakeProcessChecker = new(fakeProcessChecker)
		fakeProcessKiller    *fakeProcessKiller  = new(fakeProcessKiller)
		pureFake             *iexec.PureFake
		connectionTimeoutErr error
	)

	BeforeEach(func() {
		connectionTimeoutErr = nil
		instanceInformer = new(fakeInstanceInformer)
		logger = lagertest.NewTestLogger("process-controller")
		pureFake = iexec.NewPureFake()
	})

	JustBeforeEach(func() {
		processController = NewOSProcessController(
			logger,
			instanceInformer,
			fakeProcessChecker,
			fakeProcessKiller,
			func(instance *Instance) error {
				return errors.New("what")
			},
			func(*net.TCPAddr, time.Duration) error {
				return connectionTimeoutErr
			},
			"",
		)
		processController.exec = pureFake.Exec
	})

	itStartsARedisProcess := func(executablePath string) {
		command, args := pureFake.Exec.CommandArgsForCall(0)
		joinedArgs := strings.Join(args, " ")
		Expect(command).To(Equal(executablePath))
		Expect(joinedArgs).To(Equal("configFilePath --pidfile pidFilePath --dir instanceDataDir --logfile logFilePath"))
	}

	Describe("StartAndWaitUntilReady", func() {
		It("runs the right command to start redis", func() {
			processController.StartAndWaitUntilReady(instance, "configFilePath", "instanceDataDir", "pidFilePath", "logFilePath", time.Second*1)
			itStartsARedisProcess("redis-server")
		})

		It("returns no error", func() {
			err := processController.StartAndWaitUntilReady(instance, "", "", "", "", time.Second*1)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when the redis process fails to start", func() {
			BeforeEach(func() {
				connectionTimeoutErr = errors.New("oops")
			})

			It("returns the same error that the WaitUntilConnectableFunc returns", func() {
				err := processController.StartAndWaitUntilReady(instance, "", "", "", "", time.Second*1)
				Expect(err).To(Equal(connectionTimeoutErr))
			})
		})
	})

	Describe("StartAndWaitUntilReadyWithConfig", func() {
		Context("When using a custom redis-server executable", func() {
			It("runs the right command to start redis", func() {
				processController.RedisServerExecutablePath = "custom/path/to/redis"

				args := []string{
					"configFilePath",
					"--pidfile", "pidFilePath",
					"--dir", "instanceDataDir",
					"--logfile", "logFilePath",
				}
				processController.StartAndWaitUntilReadyWithConfig(instance, args, time.Second*1)
				itStartsARedisProcess("custom/path/to/redis")
			})
		})

		It("runs the right command to start redis", func() {
			args := []string{
				"configFilePath",
				"--pidfile", "pidFilePath",
				"--dir", "instanceDataDir",
				"--logfile", "logFilePath",
			}
			processController.StartAndWaitUntilReadyWithConfig(instance, args, time.Second*1)
			itStartsARedisProcess("redis-server")
		})

		It("returns no error", func() {
			err := processController.StartAndWaitUntilReadyWithConfig(instance, []string{}, time.Second*1)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when the redis process fails to start", func() {
			BeforeEach(func() {
				connectionTimeoutErr = errors.New("oops")
			})

			It("returns the same error that the WaitUntilConnectableFunc returns", func() {
				err := processController.StartAndWaitUntilReadyWithConfig(instance, []string{}, time.Second*1)
				Expect(err).To(Equal(connectionTimeoutErr))
			})
		})
	})

	Describe("Kill", func() {
		It("kills the correct process", func() {
			err := processController.Kill(instance)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeProcessKiller.killed).To(BeTrue())
			Expect(fakeProcessKiller.lastPidKilled).To(Equal(123))
		})
	})

	Describe("EnsureRunning", func() {
		Context("if the process is already running", func() {
			var (
				controller *OSProcessController
				log        *gbytes.Buffer
			)

			BeforeEach(func() {
				fakeProcessChecker.alive = true
			})

			JustBeforeEach(func() {
				controller = processController
				log = gbytes.NewBuffer()
				controller.Logger.RegisterSink(lager.NewWriterSink(log, lager.DEBUG))
			})

			Context("and is a redis server", func() {
				Context("and is the correct redis instance", func() {
					var err error

					JustBeforeEach(func() {
						processController.PingFunc = func(instance *Instance) error {
							return nil
						}
						err = controller.EnsureRunning(instance, "", "", "", "")
					})

					It("does not return an error", func() {
						Expect(err).NotTo(HaveOccurred())
					})

					It("logs success", func() {
						Eventually(log).Should(gbytes.Say("redis instance already running"))
					})
				})

				Context("and is not the correct redis instance", func() {
					var (
						err  error
						file *os.File
					)

					BeforeEach(func() {
						var statErr error

						file, statErr = ioutil.TempFile("/tmp", "brokerTest")
						Expect(statErr).NotTo(HaveOccurred())
					})

					JustBeforeEach(func() {
						err = controller.EnsureRunning(instance, "/my/lovely/config", "", file.Name(), "")
					})

					It("does not return an error", func() {
						Expect(err).NotTo(HaveOccurred())
					})

					It("deletes the pid file", func() {
						_, statErr := os.Stat(file.Name())
						Expect(statErr).To(HaveOccurred())
					})

					It("logs the pidfile deletion", func() {
						Eventually(log).Should(gbytes.Say("removed stale pidfile"))
					})

					It("should restart the redis-server", func() {
						ran := false
						callCount := pureFake.Exec.CommandCallCount()
						for i := 0; i < callCount; i++ {
							command, args := pureFake.Exec.CommandArgsForCall(i)
							joinedArgs := strings.Join(args, " ")
							if command == "redis-server" && strings.Contains(joinedArgs, "/my/lovely/config") {
								ran = true
								break
							}
						}
						Expect(ran).To(BeTrue())
					})

					Context("and failed to delete pidfile", func() {
						JustBeforeEach(func() {
							err = controller.EnsureRunning(instance, "", "", "Pikachu", "")
						})

						It("returns an error", func() {
							Expect(err).To(HaveOccurred())
						})

						It("logs the error", func() {
							Eventually(log).Should(gbytes.Say("failed to delete stale pidfile"))
						})
					})
				})
			})

			Context("and is not a redis server", func() {
				var (
					err  error
					file *os.File
				)

				BeforeEach(func() {
					var statErr error

					file, statErr = ioutil.TempFile("/tmp", "brokerTest")
					Expect(statErr).NotTo(HaveOccurred())
				})

				JustBeforeEach(func() {
					err = controller.EnsureRunning(instance, "/my/config", "", file.Name(), "")
				})

				It("deletes the pidfile", func() {
					_, statErr := os.Stat(file.Name())
					Expect(statErr).To(HaveOccurred())
				})

				It("does not return an error", func() {
					Expect(err).NotTo(HaveOccurred())
				})

				It("logs the pidfile deletion", func() {
					Eventually(log).Should(gbytes.Say("removed stale pidfile"))
				})

				It("should restart the redis-server", func() {
					ran := false
					callCount := pureFake.Exec.CommandCallCount()
					for i := 0; i < callCount; i++ {
						command, args := pureFake.Exec.CommandArgsForCall(i)
						joinedArgs := strings.Join(args, " ")
						if command == "redis-server" && strings.Contains(joinedArgs, "/my/config") {
							ran = true
							break
						}
					}
					Expect(ran).To(BeTrue())
				})

				Context("and failed to delete pidfile", func() {
					JustBeforeEach(func() {
						err = controller.EnsureRunning(instance, "", "", "Pikachu", "")
					})

					It("returns an error", func() {
						Expect(err).To(HaveOccurred())
					})

					It("logs the error", func() {
						Eventually(log).Should(gbytes.Say("failed to delete stale pidfile"))
					})
				})
			})
		})

		Context("if the process is not already running", func() {
			BeforeEach(func() {
				fakeProcessChecker.alive = false
			})

			It("starts it", func() {
				err := processController.EnsureRunning(instance, "configFilePath", "instanceDataDir", "pidFilePath", "logFilePath")
				Expect(err).NotTo(HaveOccurred())

				itStartsARedisProcess("redis-server")
			})

			Context("and it can not be started", func() {
				BeforeEach(func() {
					pureFake.Cmd.RunReturns(errors.New("run error"))
				})

				It("should return error", func() {
					err := processController.EnsureRunning(instance, "", "", "", "")
					Expect(err).To(HaveOccurred())
				})
			})
		})
	})
})
