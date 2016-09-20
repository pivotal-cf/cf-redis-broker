package system_test

import (
	"code.cloudfoundry.org/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf/cf-redis-broker/system"
)

var _ = Describe("System command Runner", func() {

	var commandRunner system.CommandRunner

	BeforeEach(func() {
		commandRunner = system.OSCommandRunner{
			Logger: lagertest.NewTestLogger("command-runner"),
		}
	})

	Context("When the command is valid", func() {
		It("Runs a command successfully", func() {
			err := commandRunner.Run("echo", "Hello", "World")
			Ω(err).ToNot(HaveOccurred())
		})
	})
})
