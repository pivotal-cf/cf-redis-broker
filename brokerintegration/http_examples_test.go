package brokerintegration_test

import (
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/cf-redis-broker/integration"
)

type HTTPExampleInputs struct {
	Method string
	URI    string
}

func HTTPResponseBodyShouldBeEmptyJSON(inputs *HTTPExampleInputs) {
	It("returns empty JSON", func() {
		_, body := integration.ExecuteAuthenticatedHTTPRequest(inputs.Method, inputs.URI, brokerConfig.AuthConfiguration.Username, brokerConfig.AuthConfiguration.Password)

		var parsedJSON map[string][]interface{}
		json.Unmarshal(body, &parsedJSON)

		Ω(parsedJSON).To(Equal(map[string][]interface{}{}))
	})
}

func HTTPResponseShouldContainBrokerErrorMessage(inputs *HTTPExampleInputs, expectedErrorMessage string) {
	It("returns the expected error message", func() {
		_, body := integration.ExecuteAuthenticatedHTTPRequest(inputs.Method, inputs.URI, brokerConfig.AuthConfiguration.Username, brokerConfig.AuthConfiguration.Password)

		var parsedJSON map[string]interface{}
		json.Unmarshal(body, &parsedJSON)

		errorMessage := parsedJSON["description"].(string)
		Ω(errorMessage).Should(Equal(expectedErrorMessage))
	})
}

func HTTPResponseShouldContainExpectedHTTPStatusCode(inputs *HTTPExampleInputs, expectedStatusCode int) {
	It(fmt.Sprint("returns HTTP ", expectedStatusCode), func() {
		code, _ := integration.ExecuteAuthenticatedHTTPRequest(inputs.Method, inputs.URI, brokerConfig.AuthConfiguration.Username, brokerConfig.AuthConfiguration.Password)

		Ω(code).To(Equal(expectedStatusCode))
	})
}
