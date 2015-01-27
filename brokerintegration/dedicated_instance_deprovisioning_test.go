package brokerintegration_test

import (
	"net"
	"net/http"
	"net/http/httptest"

	"code.google.com/p/go-uuid/uuid"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Deprovisioning dedicated instance", func() {
	var instanceID string
	var httpInputs HTTPExampleInputs
	var agentResponseStatus int

	Context("Deprovision running instance", func() {
		var server *httptest.Server
		var agentCalled int

		BeforeEach(func() {
			stopFakeAgent()
			switchBroker("broker.yml-one-dedicated")

			agentCalled = 0
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				Ω(r.Method).Should(Equal("DELETE"))
				Ω(r.URL.Path).Should(Equal("/"))
				w.WriteHeader(agentResponseStatus)
				agentCalled++
			})

			server = httptest.NewUnstartedServer(handler)
			listener, err := net.Listen("tcp", "127.0.0.1:9876")
			Ω(err).ShouldNot(HaveOccurred())
			server.Listener = listener
			server.Start()

			instanceID = uuid.NewRandom().String()
			httpInputs = HTTPExampleInputs{Method: "DELETE", URI: instanceURI(instanceID)}

			agentResponseStatus = http.StatusOK

			code, _ := provisionInstance(instanceID, "dedicated")
			Ω(code).Should(Equal(201))
		})

		AfterEach(func() {
			server.Close()
			switchBroker("broker.yml")
			startFakeAgent()
		})

		HTTPResponseShouldContainExpectedHTTPStatusCode(&httpInputs, 200)
		HTTPResponseBodyShouldBeEmptyJSON(&httpInputs)

		It("tells node agent to deprovision instance", func() {
			Ω(agentCalled).Should(Equal(0))
			deprovisionInstance(instanceID)
			Ω(agentCalled).Should(Equal(1))
		})

		Context("When resetting the agent fails", func() {
			BeforeEach(func() {
				agentResponseStatus = http.StatusInternalServerError
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
