package brokerintegration_test

import (
	"code.google.com/p/go-uuid/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Provision dedicated instance", func() {

	var instanceID string
	var httpInputs HTTPExampleInputs

	BeforeEach(func() {
		instanceID = uuid.NewRandom().String()
		serviceInstanceURI := "http://localhost:3000/v2/service_instances/" + instanceID
		httpInputs = HTTPExampleInputs{
			Method: "PUT",
			URI:    serviceInstanceURI,
		}
	})

	Context("when instance is created successfully", func() {
		AfterEach(func() {
			deprovisionInstance(instanceID)
		})

		It("returns 201", func() {
			status, _ := provisionInstance(instanceID, "dedicated")
			Expect(status).To(Equal(201))
		})

		It("returns empty JSON", func() {
			_, body := provisionInstance(instanceID, "dedicated")
			Expect(body).To(MatchJSON("{}"))
		})

		It("does not start a shared Redis instance", func() {
			provisionInstance(instanceID, "dedicated")
			Ω(getRedisProcessCount()).To(Equal(0))
		})
	})

	Context("when the service instance limit has been met", func() {
		BeforeEach(func() {
			provisionInstance("1", "dedicated")
			provisionInstance("2", "dedicated")
			provisionInstance("3", "dedicated")
		})

		AfterEach(func() {
			deprovisionInstance("1")
			deprovisionInstance("2")
			deprovisionInstance("3")
		})

		It("does not start a shared Redis instance", func() {
			provisionInstance("4", "dedicated")
			Ω(getRedisProcessCount()).To(Equal(0))
		})

		It("returns a 500", func() {
			statusCode, _ := provisionInstance("4", "dedicated")
			defer deprovisionInstance("4")
			Ω(statusCode).To(Equal(500))
		})

		It("returns a useful error message in the correct JSON format", func() {
			_, body := provisionInstance("4", "dedicated")
			defer deprovisionInstance("4")

			Ω(string(body)).To(MatchJSON(`{"description":"instance limit for this service has been reached"}`))
		})
	})

	Context("when the service instance already exists", func() {
		BeforeEach(func() {
			provisionInstance(instanceID, "dedicated")
		})

		AfterEach(func() {
			deprovisionInstance(instanceID)
		})

		It("should fail if we try to provision a second instance with the same ID", func() {
			numRedisProcessesBeforeExec := getRedisProcessCount()
			statusCode, body := provisionInstance(instanceID, "dedicated")
			Ω(statusCode).To(Equal(409))

			Ω(string(body)).To(MatchJSON(`{}`))
			Ω(getRedisProcessCount()).To(Equal(numRedisProcessesBeforeExec))
		})
	})
})
