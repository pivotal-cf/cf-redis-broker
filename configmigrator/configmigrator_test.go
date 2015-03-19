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

	Context("when there is no data to migrate", func() {
		It("does nothing", func() {
			err := configMigrator.Migrate()
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("when there is data to migrate", func() {
		var instanceBaseDir string

		BeforeEach(func() {
			instanceBaseDir = path.Join(redisDataDirPath, "instance1")
			err := os.Mkdir(instanceBaseDir, 0777)
			Expect(err).ToNot(HaveOccurred())
		})

		Context("and port is in redis-server.port and password is in redis-server.password", func() {
			It("deletes the redis port file and password file", func() {
				redisConfFile := path.Join(instanceBaseDir, "redis.conf")
				ioutil.WriteFile(redisConfFile, []byte("#port 63490"), 0777)

				redisPortFilePath := path.Join(instanceBaseDir, REDIS_PORT_FILENAME)
				ioutil.WriteFile(redisPortFilePath, []byte("3455"), 0777)
				redisPasswordFilePath := path.Join(instanceBaseDir, REDIS_PASSWORD_FILENAME)
				ioutil.WriteFile(redisPasswordFilePath, []byte("secret-password"), 0777)

				configMigrator.Migrate()

				_, err := os.Stat(redisPortFilePath)
				Expect(os.IsNotExist(err)).To(BeTrue())
				_, err = os.Stat(redisPasswordFilePath)
				Expect(os.IsNotExist(err)).To(BeTrue())
			})

			It("copies the port and password to redis.conf", func() {
				redisConfFile := path.Join(instanceBaseDir, "redis.conf")
				ioutil.WriteFile(redisConfFile, []byte("#port 63490"), 0777)

				redisPortFilePath := path.Join(instanceBaseDir, REDIS_PORT_FILENAME)
				ioutil.WriteFile(redisPortFilePath, []byte("3455"), 0777)

				redisPasswordFilePath := path.Join(instanceBaseDir, REDIS_PASSWORD_FILENAME)
				ioutil.WriteFile(redisPasswordFilePath, []byte("secret-password"), 0777)

				configMigrator.Migrate()

				redisConfigValues, err := redisconf.Load(redisConfFile)
				Expect(err).ToNot(HaveOccurred())
				Expect(redisConfigValues.Get("port")).To(Equal("3455"))
				Expect(redisConfigValues.Get("requirepass")).To(Equal("secret-password"))
			})

			It("does not change the other values", func() {
				redisConfFile := path.Join(instanceBaseDir, "redis.conf")
				ioutil.WriteFile(redisConfFile, []byte("foo bar"), 0777)

				configMigrator.Migrate()

				redisConfigValues, err := redisconf.Load(redisConfFile)
				Expect(err).ToNot(HaveOccurred())
				Expect(redisConfigValues.Get("foo")).To(Equal("bar"))
			})

			Context("and there are multiple instances to migrate", func() {
				It("migrates all of them", func() {
					redisConfFile := path.Join(instanceBaseDir, "redis.conf")
					ioutil.WriteFile(redisConfFile, []byte("#port 63490"), 0777)
					redisPortFilePath := path.Join(instanceBaseDir, REDIS_PORT_FILENAME)
					ioutil.WriteFile(redisPortFilePath, []byte("3455"), 0777)
					redisPasswordFilePath := path.Join(instanceBaseDir, REDIS_PASSWORD_FILENAME)
					ioutil.WriteFile(redisPasswordFilePath, []byte("secret-password"), 0777)

					instance2BaseDir := path.Join(redisDataDirPath, "instance2")
					os.Mkdir(instance2BaseDir, 0777)
					redis2ConfFile := path.Join(instance2BaseDir, "redis.conf")
					ioutil.WriteFile(redis2ConfFile, []byte("#port 63490"), 0777)
					redis2PortFilePath := path.Join(instance2BaseDir, REDIS_PORT_FILENAME)
					ioutil.WriteFile(redis2PortFilePath, []byte("9482"), 0777)
					redis2PasswordFilePath := path.Join(instance2BaseDir, REDIS_PASSWORD_FILENAME)
					ioutil.WriteFile(redis2PasswordFilePath, []byte("secret-password2"), 0777)

					configMigrator.Migrate()

					redisConfigValues, err := redisconf.Load(redisConfFile)
					Expect(err).ToNot(HaveOccurred())
					Expect(redisConfigValues.Get("port")).To(Equal("3455"))

					redisConfigValues, err = redisconf.Load(redis2ConfFile)
					Expect(err).ToNot(HaveOccurred())
					Expect(redisConfigValues.Get("port")).To(Equal("9482"))
				})
			})

			Context("and it cannot write to redis.conf", func() {
				It("returns an error", func() {
					redisConfFile := path.Join(instanceBaseDir, "redis.conf")
					ioutil.WriteFile(redisConfFile, []byte("foo bar"), 0000)

					err := configMigrator.Migrate()
					Expect(err).To(HaveOccurred())
				})
			})

			Context("and it cannot read from the redis-server.port", func() {
				It("returns an error", func() {
					redisConfFile := path.Join(instanceBaseDir, "redis.conf")
					ioutil.WriteFile(redisConfFile, []byte("#port 63490"), 0777)
					redisPortFilePath := path.Join(instanceBaseDir, REDIS_PORT_FILENAME)
					ioutil.WriteFile(redisPortFilePath, []byte("3455"), 0000)

					err := configMigrator.Migrate()
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Context("and data is already migrated", func() {
			It("does nothing", func() {
				redisConfFile := path.Join(instanceBaseDir, "redis.conf")
				ioutil.WriteFile(redisConfFile, []byte("port 6349\nrequirepass secret-password"), 0777)

				err := configMigrator.Migrate()
				Expect(err).ToNot(HaveOccurred())

				redisConfigValues, err := redisconf.Load(path.Join(instanceBaseDir, "redis.conf"))
				Expect(err).ToNot(HaveOccurred())
				Expect(redisConfigValues.Get("port")).To(Equal("6349"))
				Expect(redisConfigValues.Get("requirepass")).To(Equal("secret-password"))
			})
		})

		Context("and data is partially migrated", func() {
			It("finishes the migration for the password", func() {
				redisConfFile := path.Join(instanceBaseDir, "redis.conf")
				ioutil.WriteFile(redisConfFile, []byte("port 6349\n#requirepass INSERT_PASSWORD_HERE"), 0777)
				redisPasswordFilePath := path.Join(instanceBaseDir, REDIS_PASSWORD_FILENAME)
				ioutil.WriteFile(redisPasswordFilePath, []byte("secret-password"), 0777)

				err := configMigrator.Migrate()
				Expect(err).ToNot(HaveOccurred())

				redisConfigValues, err := redisconf.Load(path.Join(instanceBaseDir, "redis.conf"))
				Expect(err).ToNot(HaveOccurred())
				Expect(redisConfigValues.Get("port")).To(Equal("6349"))
				Expect(redisConfigValues.Get("requirepass")).To(Equal("secret-password"))
			})
			It("finishes the migration for the port", func() {
				redisConfFile := path.Join(instanceBaseDir, "redis.conf")
				ioutil.WriteFile(redisConfFile, []byte("#port 1234\nrequirepass secret-password"), 0777)
				redisPortFilePath := path.Join(instanceBaseDir, REDIS_PORT_FILENAME)
				ioutil.WriteFile(redisPortFilePath, []byte("6349"), 0777)

				err := configMigrator.Migrate()
				Expect(err).ToNot(HaveOccurred())

				redisConfigValues, err := redisconf.Load(path.Join(instanceBaseDir, "redis.conf"))
				Expect(err).ToNot(HaveOccurred())
				Expect(redisConfigValues.Get("port")).To(Equal("6349"))
				Expect(redisConfigValues.Get("requirepass")).To(Equal("secret-password"))
			})
		})

		Context("and loading of the redis.conf file is failing", func() {
			It("returns a error", func() {
				err := configMigrator.Migrate()

				Expect(err).To(HaveOccurred())
			})
		})
	})
})
