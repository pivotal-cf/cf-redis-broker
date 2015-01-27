package credentials_test

import (
	"os"
	"path/filepath"

	"github.com/pivotal-cf/cf-redis-broker/credentials"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Credentials", func() {
	var configPath string

	Context("When the conf file can be read", func() {
		BeforeEach(func() {
			configPath = filepath.Join("assets", "redis.conf")
		})

		It("returns the correct port", func() {
			creds, err := credentials.Parse(configPath)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(creds.Port).Should(Equal(0))
		})

		It("returns the correct password", func() {
			creds, err := credentials.Parse(configPath)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(creds.Password).Should(Equal("default"))
		})
	})

	Context("when the conf file is invalid", func() {
		BeforeEach(func() {
			configPath = filepath.Join("assets", "redis.conf.invalid")
		})

		It("returns the an error", func() {
			_, err := credentials.Parse(configPath)
			Ω(err.Error()).Should(Equal("strconv.ParseInt: parsing \"not_a_port\": invalid syntax"))
		})
	})

	Context("when the conf file does not exist", func() {
		BeforeEach(func() {
			configPath = filepath.Join("assets", "redis.conf.nonexistant")
		})

		It("returns the an error", func() {
			_, err := credentials.Parse(configPath)
			Ω(os.IsNotExist(err)).Should(BeTrue())
		})
	})
})
