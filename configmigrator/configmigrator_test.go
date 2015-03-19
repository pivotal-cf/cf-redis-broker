package configmigrator

import (
	"io/ioutil"
	"os"
	"path"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/cf-redis-broker/redisconf"
)

var _ = Describe("Migrating config", func() {
	Context("when port used to reside in redis.port file", func() {
		var configMigrator *ConfigMigrator
		var instanceBaseDir string

		BeforeEach(func() {
			redisDataDirPath, err := ioutil.TempDir("", "redis-data")
			Expect(err).ToNot(HaveOccurred())

			instanceBaseDir = path.Join(redisDataDirPath, "instance1")
			err = os.Mkdir(instanceBaseDir, 0777)
			Expect(err).ToNot(HaveOccurred())

			configMigrator = &ConfigMigrator{
				RedisDataDir: redisDataDirPath,
			}
		})

		It("delete the redis port file", func() {
			redisPortFilePath := path.Join(instanceBaseDir, "redis.port")
			err := ioutil.WriteFile(redisPortFilePath, []byte("3455"), 0777)
			Expect(err).ToNot(HaveOccurred())

			err = configMigrator.Migrate()
			Expect(err).ToNot(HaveOccurred())

			_, err = os.Stat(redisPortFilePath)
			Expect(os.IsNotExist(err)).To(BeTrue())
		})

		It("copies the port from redis.port to redis.conf", func() {
			redisConfFile := path.Join(instanceBaseDir, "redis.conf")
			ioutil.WriteFile(redisConfFile, []byte("#port=63490"), 0777)

			redisPortFilePath := path.Join(instanceBaseDir, "redis.port")
			ioutil.WriteFile(redisPortFilePath, []byte("3455"), 0777)

			configMigrator.Migrate()

			redisConfigValues, err := redisconf.Load(path.Join(instanceBaseDir, "redis.conf"))
			Expect(err).ToNot(HaveOccurred())

			Expect(redisConfigValues.Get("port")).To(Equal("3455"))
		})
	})
})
