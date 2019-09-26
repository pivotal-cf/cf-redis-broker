package brokerintegration_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf/cf-redis-broker/integration"
	"github.com/pivotal-cf/cf-redis-broker/integration/helpers"
)

var _ = Describe("starting the broker", func() {
	var (
		broker *gexec.Session
		config string
	)

	BeforeEach(func() {
		config = "broker.yml-colocated"
	})

	JustBeforeEach(func() {
		broker = integration.LaunchProcessWithBrokerConfig(brokerExecutablePath, config)
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
})
