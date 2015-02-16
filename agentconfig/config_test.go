package agentconfig_test

import (
	"path"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf/cf-redis-broker/agentconfig"
)

var _ = Describe("Config", func() {
	Describe("Load", func() {
		Context("When the file does not exist", func() {
			It("returns an error", func() {
				_, err := agentconfig.Load("/this/is/an/invalid/path")
				Expect(err.Error()).To(Equal("open /this/is/an/invalid/path: no such file or directory"))
			})
		})

		Context("When the file is successfully loaded", func() {
			var config *agentconfig.Config

			BeforeEach(func() {
				path, err := filepath.Abs(path.Join("assets", "agent.yml"))
				Expect(err).ToNot(HaveOccurred())

				config, err = agentconfig.Load(path)
				Expect(err).ToNot(HaveOccurred())
			})

			It("Has the correct default_conf_path", func() {
				Expect(config.DefaultConfPath).To(Equal("/default/conf/path"))
			})

			It("Has the correct conf_path", func() {
				Expect(config.ConfPath).To(Equal("/conf/path"))
			})

			It("Has the correct monit_executable_path", func() {
				Expect(config.MonitExecutablePath).To(Equal("/foo/monit"))
			})

			It("Has the correct backend_port", func() {
				Expect(config.Port).To(Equal("9876"))
			})

			It("Has the correct username and password", func() {
				Expect(config.AuthConfiguration.Username).To(Equal("admin"))
				Expect(config.AuthConfiguration.Password).To(Equal("secret"))
			})
		})
	})
})
