package brokerintegration_test

import (
	"code.google.com/p/go-uuid/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Provision dedicated instance", func() {

	var instanceID string

	BeforeEach(func() {
		instanceID = uuid.NewRandom().String()
		provisionInstance(instanceID, "dedicated")
	})

	AfterEach(func() {
		deprovisionInstance(instanceID)
	})

	Context("when the broker is restarted", func() {
		BeforeEach(func() {
			killProcess(brokerSession)
			brokerSession = buildAndLaunchBroker("broker.yml")
			Ω(portAvailable(brokerPort)).Should(BeTrue())
		})

		It("retains state", func() {
			debugInfo := getDebugInfo()

			Ω(debugInfo.Allocated.Count).Should(Equal(1))
			Ω(len(debugInfo.Allocated.Clusters)).Should(Equal(1))

			host := debugInfo.Allocated.Clusters[0].Hosts[0]
			Ω(host).Should(MatchRegexp("server[1-3]\\.127\\.0\\.0\\.1\\.xip\\.io"))

			Ω(debugInfo.Pool.Clusters).ShouldNot(ContainElement([]string{host}))
		})
	})

	Context("when the broker is restarted with a new node", func() {
		BeforeEach(func() {
			killProcess(brokerSession)
			brokerSession = buildAndLaunchBroker("broker.yml-extra-node")
			Ω(portAvailable(brokerPort)).Should(BeTrue())
		})

		AfterEach(func() {
			killProcess(brokerSession)
			brokerSession = buildAndLaunchBroker("broker.yml")
			Ω(portAvailable(brokerPort)).Should(BeTrue())
		})

		It("retains state, and adds the extra node", func() {
			debugInfo := getDebugInfo()

			Ω(debugInfo.Allocated.Count).Should(Equal(1))
			Ω(len(debugInfo.Allocated.Clusters)).Should(Equal(1))

			host := debugInfo.Allocated.Clusters[0].Hosts[0]
			Ω(host).Should(MatchRegexp("server[1-3]\\.127\\.0\\.0\\.1\\.xip\\.io"))

			Ω(debugInfo.Pool.Clusters).ShouldNot(ContainElement([]string{host}))
			Ω(debugInfo.Pool.Clusters).Should(ContainElement([]string{"server4.127.0.0.1.xip.io"}))
		})

	})
})
