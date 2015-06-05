package instance_test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/cf-redis-broker/instance"
	"github.com/pivotal-cf/cf-redis-broker/redisconf"
)

var _ = Describe("RedisConfigFinder", func() {

	Describe(".Find", func() {
		var (
			redisConfigs []instance.RedisConfig
			findErr      error
			rootPath     string
			filename     = "redis.conf"
		)

		JustBeforeEach(func() {
			finder := instance.NewRedisConfigFinder(rootPath, filename)
			redisConfigs, findErr = finder.Find()
		})

		Context("when the root exists", func() {
			var createRedisConfig = func(path, key, value string) (redisconf.Conf, error) {
				redisConfig := &redisconf.Conf{}
				redisConfig.Set(key, value)
				return *redisConfig, redisConfig.Save(path)
			}

			var (
				expectedKey   = "some-key"
				expectedValue = "some-value"
			)

			BeforeEach(func() {
				var err error
				rootPath, err = ioutil.TempDir("", "redis")
				Expect(err).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				os.Remove(rootPath)
			})

			Context("when there are no redis configurations", func() {
				It("does not return an error", func() {
					Expect(findErr).ToNot(HaveOccurred())
				})

				It("returns an empty slice", func() {
					Expect(redisConfigs).ToNot(Equal(nil))
					Expect(redisConfigs).To(HaveLen(0))
				})
			})

			Context("when there is one redis configuration", func() {
				var (
					redisConfigPath string
					redisConf       redisconf.Conf
				)

				BeforeEach(func() {
					redisConfigPath = filepath.Join(rootPath, filename)
					var err error
					redisConf, err = createRedisConfig(redisConfigPath, expectedKey, expectedValue)
					Expect(err).ToNot(HaveOccurred())
				})

				It("does not return an error", func() {
					Expect(findErr).ToNot(HaveOccurred())
				})

				It("returns the redis conf", func() {
					Expect(redisConfigs).To(ContainElement(instance.RedisConfig{
						Path: redisConfigPath,
						Conf: redisConf,
					}))
				})
			})

			Context("when the root contains multiple redis configurations", func() {
				var redisConfs map[string]redisconf.Conf

				BeforeEach(func() {
					redisConfs = map[string]redisconf.Conf{}

					for i := 0; i < 3; i++ {
						path, err := ioutil.TempDir(rootPath, "instance")
						Expect(err).ToNot(HaveOccurred())

						path = filepath.Join(path, filename)

						redisConfs[path], err = createRedisConfig(path, expectedKey, expectedValue)
						Expect(err).ToNot(HaveOccurred())
					}
				})

				It("does not return an error", func() {
					Expect(findErr).ToNot(HaveOccurred())
				})

				It("returns all redis configs", func() {
					for path, conf := range redisConfs {
						Expect(redisConfigs).To(ContainElement(
							instance.RedisConfig{
								Conf: conf,
								Path: path,
							},
						))
					}
				})
			})
		})

		Context("when the root does not exist", func() {
			BeforeEach(func() {
				rootPath = "non-existing"
			})

			It("returns an error", func() {
				Expect(findErr).To(HaveOccurred())
			})
		})
	})
})
