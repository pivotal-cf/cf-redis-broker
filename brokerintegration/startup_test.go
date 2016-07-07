package brokerintegration_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf/cf-redis-broker/integration"
	"github.com/pivotal-cf/cf-redis-broker/integration/helpers"
)

var _ = Describe("starting the broker", func() {
	var broker *gexec.Session

	BeforeEach(func() {
		broker = integration.LaunchProcessWithBrokerConfig(brokerExecutablePath, "broker.yml-colocated")
	})

	AfterEach(func() {
		helpers.KillProcess(broker)
	})

	It("logs a startup message", func() {
		Eventually(broker.Out).Should(gbytes.Say("Starting CF Redis broker"))
	})

	It("logs that it has identified zero shared instances", func() {
		Eventually(broker.Out).Should(gbytes.Say("0 shared Redis instances found"))
	})

	It("logs that it has identified zero dedicated instances", func() {
		statefilePath := brokerConfig.RedisConfiguration.Dedicated.StatefilePath
		Eventually(broker.Out).Should(gbytes.Say(fmt.Sprintf("statefile %s not found, generating instead", statefilePath)))
	})

	It("logs that it has identified zero dedicated instances", func() {
		Eventually(broker.Out).Should(gbytes.Say("0 dedicated Redis instances found"))
	})
})
