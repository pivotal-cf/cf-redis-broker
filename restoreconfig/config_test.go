package restoreconfig_test

import (
	"io/ioutil"
	"os"
	"path"

	"github.com/pivotal-cf/cf-redis-broker/restoreconfig"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	var restoreConfig restoreconfig.Config

	Context("Loads values from restore_config.yml", func() {
		BeforeEach(func() {
			var err error
			restoreConfig, err = restoreconfig.Load(path.Join("assets", "restore-dedicated.yml"))
			Expect(err).ToNot(HaveOccurred())
		})

		It("Reads the monit executable", func() {
			Expect(restoreConfig.MonitExecutablePath).To(Equal("/path/to/monit/file"))
		})

		It("Reads the redis data directory", func() {
			Expect(restoreConfig.RedisDataDirectory).To(Equal("/tmp/redis/data"))
		})

		It("Reads the pidfile directory", func() {
			Expect(restoreConfig.PidfileDirectory).To(Equal("/tmp/pidfiles"))
		})

		It("Reads the RedisServerExecutablePath", func() {
			Expect(restoreConfig.RedisServerExecutablePath).To(Equal("/path/to/redis"))
		})

		It("Reads the StartRedisTimeoutSeconds", func() {
			Expect(restoreConfig.StartRedisTimeoutSeconds).To(Equal(123))
		})

		It("Reads the dedicated instance flag", func() {
			Expect(restoreConfig.DedicatedInstance).To(BeTrue())
		})
	})

	Context("Shared vm", func() {
		BeforeEach(func() {
			var err error
			restoreConfig, err = restoreconfig.Load(path.Join("assets", "restore-shared.yml"))
			Expect(err).ToNot(HaveOccurred())
		})

		Describe("#InstancePid", func() {
			var pidfilePath string

			BeforeEach(func() {
				err := os.MkdirAll(restoreConfig.PidfileDirectory, 0777)
				Expect(err).NotTo(HaveOccurred())

				pidfilePath = path.Join(restoreConfig.PidfileDirectory, "instance-id.pid")
				err = ioutil.WriteFile(pidfilePath, []byte("1234"), 0777)
				Expect(err).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				if pidfilePath == "" {
					err := os.Remove(pidfilePath)
					Expect(err).NotTo(HaveOccurred())
				}
			})

			It("Returns instance PID", func() {
				Expect(restoreConfig.InstancePid("instance-id")).To(Equal(1234))
			})
		})

		Describe("#InstancePidFilePath", func() {
			It("Returns the instance PID file path", func() {
				expected := "/tmp/pidfiles/instance-id.pid"
				Expect(restoreConfig.InstancePidFilePath("instance-id")).To(Equal(expected))
			})
		})

		Describe("#InstanceDataDir", func() {
			It("Returns the instance data directory", func() {
				Expect(restoreConfig.InstanceDataDir("instance-id")).To(Equal("/tmp/redis/data/instance-id/db"))
			})
		})
	})

	Context("Dedicated vm", func() {
		BeforeEach(func() {
			var err error
			restoreConfig, err = restoreconfig.Load(path.Join("assets", "restore-dedicated.yml"))
			Expect(err).ToNot(HaveOccurred())
		})

		Describe("#InstancePid", func() {
			var pidfilePath string

			BeforeEach(func() {
				err := os.MkdirAll(restoreConfig.PidfileDirectory, 0777)
				Expect(err).NotTo(HaveOccurred())

				pidfilePath = path.Join(restoreConfig.PidfileDirectory, "redis.pid")
				err = ioutil.WriteFile(pidfilePath, []byte("1234"), 0777)
				Expect(err).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				if pidfilePath == "" {
					err := os.Remove(pidfilePath)
					Expect(err).NotTo(HaveOccurred())
				}
			})

			It("Returns instance PID", func() {
				Expect(restoreConfig.InstancePid("instance-id")).To(Equal(1234))
			})
		})

		Describe("#InstancePidFilePath", func() {
			It("Returns the instance PID file path", func() {
				expected := "/tmp/pidfiles/redis.pid"
				Expect(restoreConfig.InstancePidFilePath("instance-id")).To(Equal(expected))
			})
		})

		Describe("#InstanceDataDir", func() {
			It("Returns the instance data directory", func() {
				Expect(restoreConfig.InstanceDataDir("instance-id")).To(Equal("/tmp/redis/data"))
			})
		})
	})
})
