package brokerintegration_test

import (
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type HTTPExampleInputs struct {
	Method string
	URI    string
}

func HTTPResponseBodyShouldBeEmptyJSON(inputs *HTTPExampleInputs) {
	It("returns empty JSON", func() {
		_, body := executeAuthenticatedHTTPRequest(inputs.Method, inputs.URI)

		var parsedJSON map[string][]interface{}
		json.Unmarshal(body, &parsedJSON)

		Ω(parsedJSON).To(Equal(map[string][]interface{}{}))
	})
}

func HTTPResponseShouldContainBrokerErrorMessage(inputs *HTTPExampleInputs, expectedErrorMessage string) {
	It("returns the expected error message", func() {
		_, body := executeAuthenticatedHTTPRequest(inputs.Method, inputs.URI)

		var parsedJSON map[string]interface{}
		json.Unmarshal(body, &parsedJSON)

		errorMessage := parsedJSON["description"].(string)
		Ω(errorMessage).Should(Equal(expectedErrorMessage))
	})
}

func HTTPResponseShouldContainExpectedHTTPStatusCode(inputs *HTTPExampleInputs, expectedStatusCode int) {
	It(fmt.Sprint("returns HTTP ", expectedStatusCode), func() {
		code, _ := executeAuthenticatedHTTPRequest(inputs.Method, inputs.URI)

		Ω(code).To(Equal(expectedStatusCode))
	})
}
