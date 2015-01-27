package brokerintegration_test

import (
	"code.google.com/p/go-uuid/uuid"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Deprovisioning shared instance", func() {

	var instanceID string
	var httpInputs HTTPExampleInputs

	Context("Deprovision running instance", func() {
		BeforeEach(func() {

			instanceID = uuid.NewRandom().String()
			httpInputs = HTTPExampleInputs{Method: "DELETE", URI: instanceURI(instanceID)}

			code, _ := provisionInstance(instanceID, "shared")
			Ω(code).To(Equal(201))
		})

		HTTPResponseShouldContainExpectedHTTPStatusCode(&httpInputs, 200)
		HTTPResponseBodyShouldBeEmptyJSON(&httpInputs)

		It("stops the redis process", func() {

			Ω(getRedisProcessCount()).To(Equal(1))

			deprovisionInstance(instanceID)

			// leave time for process to shutdown gracefully
			waitUntilNoRunningRedis(10.0)
			Ω(getRedisProcessCount()).To(Equal(0))
		})
	})

	Context("Deprovision missing instance", func() {
		It("should fail if the instance being deprovisioned is missing", func() {
			missingInstanceID := uuid.NewRandom().String()
			code, _ := deprovisionInstance(missingInstanceID)
			Ω(code).To(Equal(410))
		})
	})
})
