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

	BeforeEach(func() {
		var err error
		restoreConfig, err = restoreconfig.Load(path.Join("assets", "restore.yml"))
		Expect(err).ToNot(HaveOccurred())
	})

	Context("Loads values from restore_config.yml", func() {
		It("Reads the monit executable", func() {
			Expect(restoreConfig.MonitExecutablePath).To(Equal("/path/to/monit/file"))
		})

		It("Reads the redis data directory", func() {
			Expect(restoreConfig.RedisDataDirectory).To(Equal("/tmp/redis/data"))
		})

		It("Reads the RedisServerExecutablePath", func() {
			Expect(restoreConfig.RedisServerExecutablePath).To(Equal("/path/to/redis"))
		})

		It("Reads the StartRedisTimeoutSeconds", func() {
			Expect(restoreConfig.StartRedisTimeoutSeconds).To(Equal(123))
		})
	})

	Context("Shared vm", func() {
		Describe("#InstancePid", func() {
			It("Returns instance PID", func() {
				instanceDataDir := path.Join(restoreConfig.RedisDataDirectory, "instance-id")
				os.MkdirAll(instanceDataDir, 0777)
				err := ioutil.WriteFile(path.Join(instanceDataDir, "redis-server.pid"), []byte("1234"), 0777)
				Expect(err).ToNot(HaveOccurred())
				Expect(restoreConfig.InstancePid("instance-id")).To(Equal(1234))
			})
		})

		Describe("#InstancePidFilePath", func() {
			It("Returns the instance PID file path", func() {
				Expect(restoreConfig.InstancePidFilePath("instance-id")).To(Equal("/tmp/redis/data/instance-id/redis-server.pid"))
			})
		})

		Describe("#InstanceDataDir", func() {
			It("Returns the instance data directory", func() {
				Expect(restoreConfig.InstanceDataDir("instance-id")).To(Equal("/tmp/redis/data/instance-id/db"))
			})
		})
	})
})
