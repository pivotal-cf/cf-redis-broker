package brokerintegration_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pborman/uuid"
)

var _ = Describe("Provision dedicated instance", func() {

	var instanceID string

	BeforeEach(func() {
		instanceID = uuid.NewRandom().String()
	})

	Context("when instance is created successfully", func() {
		AfterEach(func() {
			brokerClient.DeprovisionInstance(instanceID)
		})

		It("returns 201", func() {
			status, _ := brokerClient.ProvisionInstance(instanceID, "dedicated")
			Expect(status).To(Equal(201))
		})

		It("returns empty JSON", func() {
			_, body := brokerClient.ProvisionInstance(instanceID, "dedicated")
			Expect(body).To(MatchJSON("{}"))
		})

		It("does not start a shared Redis instance", func() {
			brokerClient.ProvisionInstance(instanceID, "dedicated")
			Ω(getRedisProcessCount()).To(Equal(0))
		})
	})

	Context("when the service instance limit has been met", func() {
		BeforeEach(func() {
			brokerClient.ProvisionInstance("1", "dedicated")
			brokerClient.ProvisionInstance("2", "dedicated")
			brokerClient.ProvisionInstance("3", "dedicated")
		})

		AfterEach(func() {
			brokerClient.DeprovisionInstance("1")
			brokerClient.DeprovisionInstance("2")
			brokerClient.DeprovisionInstance("3")
		})

		It("does not start a shared Redis instance", func() {
			brokerClient.ProvisionInstance("4", "dedicated")
			Ω(getRedisProcessCount()).To(Equal(0))
		})

		It("returns a 500", func() {
			statusCode, _ := brokerClient.ProvisionInstance("4", "dedicated")
			defer brokerClient.DeprovisionInstance("4")
			Ω(statusCode).To(Equal(500))
		})

		It("returns a useful error message in the correct JSON format", func() {
			_, body := brokerClient.ProvisionInstance("4", "dedicated")
			defer brokerClient.DeprovisionInstance("4")

			Ω(string(body)).To(MatchJSON(`{"description":"instance limit for this service has been reached"}`))
		})
	})

	Context("when the service instance already exists", func() {
		BeforeEach(func() {
			brokerClient.ProvisionInstance(instanceID, "dedicated")
		})

		AfterEach(func() {
			brokerClient.DeprovisionInstance(instanceID)
		})

		It("should fail if we try to provision a second instance with the same ID", func() {
			numRedisProcessesBeforeExec := getRedisProcessCount()
			statusCode, body := brokerClient.ProvisionInstance(instanceID, "dedicated")
			Ω(statusCode).To(Equal(409))

			Ω(string(body)).To(MatchJSON(`{}`))
			Ω(getRedisProcessCount()).To(Equal(numRedisProcessesBeforeExec))
		})
	})
})
