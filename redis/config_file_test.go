package redis_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/cf-redis-broker/redis"
)

var _ = Describe("ConfigFile", func() {
	var instance *redis.Instance
	var defaultConfigFilePath string
	var testConfigFilePath string
	var testDir string

	BeforeEach(func() {
		var err error
		defaultConfigFilePath, err = filepath.Abs(path.Join("assets", "redis.conf"))
		Ω(err).ToNot(HaveOccurred())
		testDir, err = ioutil.TempDir("", "config_file_test")
		Ω(err).ToNot(HaveOccurred())
		testConfigFilePath = path.Join(testDir, "redis.conf")

		instance = &redis.Instance{
			ID: "an-instance-id",
		}
	})

	AfterEach(func() {
		os.RemoveAll(testDir)
	})

	Describe("SaveRedisConfAdditions", func() {
		It("writes the config to a file", func() {
			err := redis.SaveRedisConfAdditions(defaultConfigFilePath, testConfigFilePath, instance)
			Ω(err).ToNot(HaveOccurred())

			_, err = os.Stat(testConfigFilePath)
			Ω(err).ToNot(HaveOccurred())
		})

		It("sets the config file permissons to 0644", func() {
			err := redis.SaveRedisConfAdditions(defaultConfigFilePath, testConfigFilePath, instance)
			Ω(err).ToNot(HaveOccurred())

			fileInfo, err := os.Stat(testConfigFilePath)
			Ω(err).ToNot(HaveOccurred())

			Ω(fileInfo.Mode()).To(Equal(os.FileMode(0644)))
		})

		It("writes the syslog configuration", func() {
			err := redis.SaveRedisConfAdditions(defaultConfigFilePath, testConfigFilePath, instance)
			Ω(err).ToNot(HaveOccurred())

			actualConfig, err := ioutil.ReadFile(testConfigFilePath)
			Ω(err).ToNot(HaveOccurred())

			Ω(string(actualConfig)).Should(ContainSubstring(`syslog-enabled yes`))
			Ω(string(actualConfig)).Should(ContainSubstring(fmt.Sprintf(`syslog-ident redis-server-%s`, instance.ID)))
			Ω(string(actualConfig)).Should(ContainSubstring(`syslog-facility local0`))
		})
	})
})
