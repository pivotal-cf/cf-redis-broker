package brokerintegration_test

import (
	"net/http"

	"code.google.com/p/go-uuid/uuid"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Deprovisioning dedicated instance", func() {
	var instanceID string
	var httpInputs HTTPExampleInputs

	Context("Deprovision running instance", func() {
		BeforeEach(func() {
			instanceID = uuid.NewRandom().String()
			httpInputs = HTTPExampleInputs{Method: "DELETE", URI: instanceURI(instanceID)}

			code, _ := provisionInstance(instanceID, "dedicated")
			Ω(code).Should(Equal(201))
		})

		HTTPResponseShouldContainExpectedHTTPStatusCode(&httpInputs, 200)
		HTTPResponseBodyShouldBeEmptyJSON(&httpInputs)

		It("tells node agent to deprovision instance", func() {
			agentRequests = []*http.Request{}
			deprovisionInstance(instanceID)
			Expect(agentRequests).To(HaveLen(1))
			Expect(agentRequests[0].Method).To(Equal("DELETE"))
			Expect(agentRequests[0].URL.Path).To(Equal("/"))
		})

		Context("When resetting the agent fails", func() {
			BeforeEach(func() {
				agentResponseStatus = http.StatusInternalServerError
			})

			AfterEach(func() {
				agentResponseStatus = http.StatusOK
				deprovisionInstance(instanceID)
			})

			It("returns failing error code", func() {
				code, _ := deprovisionInstance(instanceID)
				Ω(code).Should(Equal(500))
			})

			It("does not deallocate the instance", func() {
				intialAllocatedCount := getDebugInfo().Allocated.Count
				deprovisionInstance(instanceID)
				finalAllocatedCount := getDebugInfo().Allocated.Count
				Ω(finalAllocatedCount).To(Equal(intialAllocatedCount))
			})
		})
	})

	Context("Deprovision missing instance", func() {
		It("should fail if the instance being deprovisioned is missing", func() {
			missingInstanceID := uuid.NewRandom().String()
			code, _ := deprovisionInstance(missingInstanceID)
			Ω(code).Should(Equal(410))
		})
	})
})
