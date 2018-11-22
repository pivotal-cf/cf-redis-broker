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
	Body   []byte
}

func NewPUTRequest(uri, serviceId, planId string) HTTPExampleInputs {
	payload := struct {
		PlanID    string `json:"plan_id"`
		ServiceID string `json:"service_id"`
	}{
		PlanID:    planId,
		ServiceID: serviceId,
	}

	payloadBytes, err := json.Marshal(&payload)
	if err != nil {
		panic("unable to marshal the payload")
	}

	return HTTPExampleInputs{
		Method: "PUT",
		URI:    uri,
		Body:   payloadBytes,
	}
}

func HTTPResponseBodyShouldBeEmptyJSON(inputs *HTTPExampleInputs) {
	It("returns empty JSON", func() {
		_, body := integration.ExecuteAuthenticatedHTTPRequestWithBody(inputs.Method, inputs.URI, brokerConfig.AuthConfiguration.Username, brokerConfig.AuthConfiguration.Password, inputs.Body)

		var parsedJSON map[string][]interface{}
		json.Unmarshal(body, &parsedJSON)

		Ω(parsedJSON).To(Equal(map[string][]interface{}{}))
	})
}

func HTTPResponseShouldContainBrokerErrorMessage(inputs *HTTPExampleInputs, expectedErrorMessage string) {
	It("returns the expected error message", func() {
		_, body := integration.ExecuteAuthenticatedHTTPRequestWithBody(inputs.Method, inputs.URI, brokerConfig.AuthConfiguration.Username, brokerConfig.AuthConfiguration.Password, inputs.Body)

		var parsedJSON map[string]interface{}
		json.Unmarshal(body, &parsedJSON)

		errorMessage := parsedJSON["description"].(string)
		Ω(errorMessage).Should(Equal(expectedErrorMessage))
	})
}

func HTTPResponseShouldContainExpectedHTTPStatusCode(inputs *HTTPExampleInputs, expectedStatusCode int) {
	It(fmt.Sprint("returns HTTP ", expectedStatusCode), func() {
		code, _ := integration.ExecuteAuthenticatedHTTPRequestWithBody(inputs.Method, inputs.URI, brokerConfig.AuthConfiguration.Username, brokerConfig.AuthConfiguration.Password, inputs.Body)

		Ω(code).To(Equal(expectedStatusCode))
	})
}
