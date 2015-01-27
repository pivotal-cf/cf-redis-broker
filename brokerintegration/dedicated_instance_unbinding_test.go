package brokerintegration_test

import (
	"fmt"
	"net/http"

	"code.google.com/p/go-uuid/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Dedicated instance unbinding", func() {
	var instanceID string
	var bindingID string

	BeforeEach(func() {
		instanceID = uuid.NewRandom().String()
		bindingID = uuid.NewRandom().String()

		code, _ := provisionInstance(instanceID, "dedicated")
		Ω(code).Should(Equal(201))

		status, _ := bindInstance(instanceID, bindingID)
		Ω(status).Should(Equal(http.StatusCreated))
	})

	AfterEach(func() {
		deprovisionInstance(instanceID)
	})

	It("should respond correctly", func() {

		validURI := fmt.Sprintf("http://localhost:3000/v2/service_instances/%s/service_bindings/%s", instanceID, bindingID)
		invalidURI := fmt.Sprintf("http://localhost:3000/v2/service_instances/%s/service_bindings/%s", "NON-EXISTANT", bindingID)

		code, body := executeAuthenticatedHTTPRequest("DELETE", validURI)
		Ω(code).Should(Equal(200))
		Ω(body).Should(MatchJSON("{}"))

		code, body = executeAuthenticatedHTTPRequest("DELETE", validURI)
		Ω(code).To(Equal(410))
		Ω(body).Should(MatchJSON("{}"))

		code, body = executeAuthenticatedHTTPRequest("DELETE", invalidURI)
		Ω(code).To(Equal(404))
		Ω(body).Should(MatchJSON("{}"))
	})
})
