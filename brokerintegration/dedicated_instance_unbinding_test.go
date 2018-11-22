package brokerintegration_test

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pborman/uuid"
)

var _ = Describe("Dedicated instance unbinding", func() {
	var instanceID string
	var bindingID string

	BeforeEach(func() {
		instanceID = uuid.NewRandom().String()
		bindingID = uuid.NewRandom().String()

		code, _ := brokerClient.ProvisionInstance(instanceID, "dedicated")
		Ω(code).Should(Equal(201))

		status, _ := brokerClient.BindInstance(instanceID, bindingID, "dedicated")
		Ω(status).Should(Equal(http.StatusCreated))
	})

	AfterEach(func() {
		brokerClient.DeprovisionInstance(instanceID, "dedicated")
	})

	It("should respond correctly", func() {
		code, body := brokerClient.UnbindInstance(instanceID, bindingID, "dedicated")
		Ω(code).Should(Equal(200))
		Ω(body).Should(MatchJSON("{}"))

		code, body = brokerClient.UnbindInstance(instanceID, bindingID, "dedicated")
		Ω(code).To(Equal(410))
		Ω(body).Should(MatchJSON("{}"))

		code, body = brokerClient.UnbindInstance("NON-EXISTANT", bindingID, "dedicated")
		Ω(code).To(Equal(410))
		Ω(body).Should(MatchJSON("{}"))
	})
})
