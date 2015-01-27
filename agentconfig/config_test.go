package agentconfig_test

import (
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf/cf-redis-broker/agentconfig"
)

var _ = Describe("Config", func() {
	Describe("Load", func() {
		var file *os.File
		var path string

		BeforeEach(func() {
			var err error
			file, err = ioutil.TempFile("", "config.yml")
			Expect(err).ToNot(HaveOccurred())
			path = file.Name()
		})

		AfterEach(func() {
			file.Close()
		})

		Context("When the file does not exist", func() {
			It("returns an error", func() {
				_, err := agentconfig.Load("/this/is/an/invalid/path")
				Expect(err.Error()).To(Equal("open /this/is/an/invalid/path: no such file or directory"))
			})
		})

		Context("When the file is successfully loaded", func() {
			var config *agentconfig.Config
			contents := "" +
				"default_conf_path: /default/conf/path\n" +
				"conf_path: /conf/path\n" +
				"monit_executable_path: /foo/monit\n"

			BeforeEach(func() {
				_, err := file.WriteString(contents)
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
		})
	})
})
