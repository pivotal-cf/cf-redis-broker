package brokerintegration_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pborman/uuid"
	"github.com/pivotal-cf/cf-redis-broker/debug"
)

var _ = Describe("Dedicated instance binding", func() {

	var instanceID string
	var bindingID string
	var httpInputs HTTPExampleInputs

	BeforeEach(func() {
		instanceID = uuid.NewRandom().String()
		bindingID = uuid.NewRandom().String()
		httpInputs = HTTPExampleInputs{
			Method: "PUT",
			URI:    brokerClient.BindingURI(instanceID, bindingID),
		}
	})

	Context("when the instance already exists", func() {
		BeforeEach(func() {
			code, _ := brokerClient.ProvisionInstance(instanceID, "dedicated")
			Ω(code).To(Equal(201))
		})

		AfterEach(func() {
			brokerClient.DeprovisionInstance(instanceID)
		})

		HTTPResponseShouldContainExpectedHTTPStatusCode(&httpInputs, 201)

		Describe("the credentials", func() {

			var credentials map[string]interface{}
			var debugInfo debug.Info

			BeforeEach(func() {
				debugInfo = getDebugInfo()

				_, body := brokerClient.BindInstance(instanceID, bindingID)

				parsedJSON := struct {
					Credentials map[string]interface{} `json:"credentials"`
				}{}
				json.Unmarshal(body, &parsedJSON)

				credentials = parsedJSON.Credentials
			})

			It("has the correct host", func() {
				Expect(credentials["host"]).To(Equal(debugInfo.Allocated.Clusters[0].Hosts[0]))
			})

			It("has no password", func() {
				Expect(credentials["password"]).To(Equal("super-secret"))
			})

			It("has the default Redis port", func() {
				Expect(credentials["port"]).To(Equal(float64(12345)))
			})
		})
	})

	Context("when the instance does not already exist", func() {
		HTTPResponseShouldContainExpectedHTTPStatusCode(&httpInputs, 404)

		HTTPResponseShouldContainBrokerErrorMessage(&httpInputs, "instance does not exist")
	})
})
