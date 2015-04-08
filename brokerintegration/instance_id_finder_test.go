package brokerintegration_test

import (
	"encoding/json"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Finding Instance IDs for Hosts", func() {
	Context("when an instance is provisioned", func() {
		AfterEach(func() {
			status, _ := brokerClient.DeprovisionInstance("SOME-GUID")
			Ω(status).Should(Equal(200))
		})

		It("returns the instance ID from the host", func() {
			instanceID := "SOME-GUID"
			brokerClient.ProvisionInstance(instanceID, "dedicated")
			_, bindingResponse := brokerClient.BindInstance(instanceID, "someBindingID")
			host := getHostFrom(bindingResponse)

			code, body := brokerClient.InstanceIDFromHost(host)

			Ω(code).Should(Equal(200))
			Ω(string(body)).Should(Equal(`{"instance_id":"SOME-GUID"}`))
		})
	})

	Context("when basic auth credentials are incorrect", func() {
		It("returns 401 Unauthorized", func() {
			code, _ := executeHTTPRequest("GET", "http://localhost:3000/instance?host=foo")
			Ω(code).Should(Equal(http.StatusUnauthorized))
		})

		It("does not return the debug information", func() {
			_, bodyBytes := executeHTTPRequest("GET", "http://localhost:3000/instance?host=foo")
			body := string(bodyBytes)
			Ω(body).Should(Equal("Not Authorized\n"))
		})
	})
})

func getHostFrom(bindingResponse []byte) string {
	var parsedJSON map[string]interface{}
	json.Unmarshal(bindingResponse, &parsedJSON)

	credentials := parsedJSON["credentials"].(map[string]interface{})
	return credentials["host"].(string)
}
