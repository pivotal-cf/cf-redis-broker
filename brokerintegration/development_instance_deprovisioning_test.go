package brokerintegration_test

import (
	"github.com/pborman/uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Deprovisioning shared instance", func() {

	var instanceID string
	var httpInputs HTTPExampleInputs

	Context("Deprovision running instance", func() {
		BeforeEach(func() {

			instanceID = uuid.NewRandom().String()
			httpInputs = HTTPExampleInputs{Method: "DELETE",
				URI: brokerClient.InstanceURI(instanceID) + "?plan_id=my-plan&service_id=my-service"}

			code, _ := brokerClient.ProvisionInstance(instanceID, "shared")
			Ω(code).To(Equal(201))
		})

		HTTPResponseShouldContainExpectedHTTPStatusCode(&httpInputs, 200)
		HTTPResponseBodyShouldBeEmptyJSON(&httpInputs)

		It("stops the redis process", func() {

			Ω(getRedisProcessCount()).To(Equal(1))

			brokerClient.DeprovisionInstance(instanceID, "shared")
			Ω(getRedisProcessCount()).To(Equal(0))
		})
	})

	Context("Deprovision missing instance", func() {
		It("should fail if the instance being deprovisioned is missing", func() {
			missingInstanceID := uuid.NewRandom().String()
			code, _ := brokerClient.DeprovisionInstance(missingInstanceID, "shared")
			Ω(code).To(Equal(410))
		})
	})
})
