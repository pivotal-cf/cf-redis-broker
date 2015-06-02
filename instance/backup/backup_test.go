package backup_test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/cf-redis-broker/instance/backup"
	"github.com/pivotal-cf/cf-redis-broker/redisconf"
)

var _ = Describe("Backup", func() {

	Describe(".LoadRedisConfigs", func() {
		var (
			loadedConfigs  []redisconf.Conf
			loadErr        error
			configRoot     string
			configFilename = "redis.conf"
		)

		JustBeforeEach(func() {
			loadedConfigs, loadErr = backup.LoadRedisConfigs(configRoot, configFilename)
		})

		Context("when the root exists", func() {
			var createRedisConfig = func(path, key, value string) error {
				redisConfig := &redisconf.Conf{}
				redisConfig.Set(key, value)
				return redisConfig.Save(path)
			}

			var (
				expectedKey   = "some-key"
				expectedValue = "some-value"
			)

			BeforeEach(func() {
				var err error
				configRoot, err = ioutil.TempDir("", "redis")
				Expect(err).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				os.Remove(configRoot)
			})

			Context("when there are no redis configurations", func() {
				It("does not return an error", func() {
					Expect(loadErr).ToNot(HaveOccurred())
				})

				It("returns an empty slice", func() {
					Expect(loadedConfigs).ToNot(Equal(nil))
					Expect(loadedConfigs).To(HaveLen(0))
				})
			})

			Context("when there is one redis configuration", func() {
				BeforeEach(func() {
					redisConfigPath := filepath.Join(configRoot, configFilename)
					err := createRedisConfig(redisConfigPath, expectedKey, expectedValue)
					Expect(err).ToNot(HaveOccurred())
				})

				It("does not return an error", func() {
					Expect(loadErr).ToNot(HaveOccurred())
				})

				It("returns the redis conf", func() {
					Expect(loadedConfigs).To(HaveLen(1))
					Expect(loadedConfigs[0].Get(expectedKey)).To(Equal(expectedValue))
				})
			})

			Context("when the root contains multiple redis configurations", func() {
				BeforeEach(func() {
					for i := 0; i < 3; i++ {
						path, err := ioutil.TempDir(configRoot, "instance")
						Expect(err).ToNot(HaveOccurred())

						path = filepath.Join(path, configFilename)

						err = createRedisConfig(path, expectedKey, expectedValue)
						Expect(err).ToNot(HaveOccurred())
					}
				})

				It("does not return an error", func() {
					Expect(loadErr).ToNot(HaveOccurred())
				})

				It("returns all redis configs", func() {
					Expect(loadedConfigs).To(HaveLen(3))

					for _, config := range loadedConfigs {
						Expect(config.Get(expectedKey)).To(Equal(expectedValue))
					}
				})
			})
		})

		Context("when the root does not exist", func() {
			BeforeEach(func() {
				configRoot = "non-existing"
			})

			It("returns an error", func() {
				Expect(loadErr).To(HaveOccurred())
			})
		})
	})
})
