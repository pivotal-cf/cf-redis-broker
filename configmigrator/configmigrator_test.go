package configmigrator

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/cf-redis-broker/redisconf"
)

var _ = Describe("Migrating config", func() {

	var configMigrator *ConfigMigrator
	var redisDataDirPath string

	BeforeEach(func() {
		var err error
		redisDataDirPath, err = ioutil.TempDir("", "redis-data")
		Expect(err).ToNot(HaveOccurred())

		configMigrator = &ConfigMigrator{
			RedisDataDir: redisDataDirPath,
		}
	})

	Context("when there is data to migrate", func() {
		var instanceBaseDir string

		BeforeEach(func() {
			instanceBaseDir = path.Join(redisDataDirPath, "instance1")
			err := os.Mkdir(instanceBaseDir, 0777)
			Expect(err).ToNot(HaveOccurred())
		})

		Context("and port is in redis.port", func() {
			It("deletes the redis port file", func() {
				redisConfFile := path.Join(instanceBaseDir, "redis.conf")
				ioutil.WriteFile(redisConfFile, []byte("#port 63490"), 0777)

				redisPortFilePath := path.Join(instanceBaseDir, REDIS_PORT_FILENAME)
				ioutil.WriteFile(redisPortFilePath, []byte("3455"), 0777)

				configMigrator.Migrate()

				_, err := os.Stat(redisPortFilePath)
				Expect(os.IsNotExist(err)).To(BeTrue())
			})

			It("copies the port from redis.port to redis.conf", func() {
				redisConfFile := path.Join(instanceBaseDir, "redis.conf")
				ioutil.WriteFile(redisConfFile, []byte("#port 63490"), 0777)

				redisPortFilePath := path.Join(instanceBaseDir, REDIS_PORT_FILENAME)
				ioutil.WriteFile(redisPortFilePath, []byte("3455"), 0777)

				configMigrator.Migrate()

				redisConfigValues, err := redisconf.Load(path.Join(instanceBaseDir, "redis.conf"))
				Expect(err).ToNot(HaveOccurred())
				Expect(redisConfigValues.Get("port")).To(Equal("3455"))
			})

			It("does not change the other values", func() {
				redisConfFile := path.Join(instanceBaseDir, "redis.conf")
				ioutil.WriteFile(redisConfFile, []byte("foo bar"), 0777)

				configMigrator.Migrate()

				redisConfigValues, err := redisconf.Load(path.Join(instanceBaseDir, "redis.conf"))
				Expect(err).ToNot(HaveOccurred())
				Expect(redisConfigValues.Get("foo")).To(Equal("bar"))
			})
		})

		Context("and port is in redis.conf", func() {
			It("does nothing", func() {
				redisConfFile := path.Join(instanceBaseDir, "redis.conf")
				ioutil.WriteFile(redisConfFile, []byte("port 6349"), 0777)

				configMigrator.Migrate()

				redisConfigValues, err := redisconf.Load(path.Join(instanceBaseDir, "redis.conf"))
				Expect(err).ToNot(HaveOccurred())
				Expect(redisConfigValues.Get("port")).To(Equal("6349"))
			})
		})

		Context("and loading of the redis.conf file is failing", func() {
			It("returns a error", func() {
				redisConfFile := path.Join(instanceBaseDir, "redis.conf")
				err := configMigrator.Migrate()

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(fmt.Sprintf("open %s: no such file or directory", redisConfFile)))
			})
		})
	})
})
