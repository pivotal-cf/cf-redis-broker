package brokerintegration_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Shared instance unbinding", func() {

	validInputs := &HTTPExampleInputs{Method: "DELETE", URI: "http://localhost:3000/v2/service_instances/foo/service_bindings/bar?plan_id=my-plan&service_id=my-service-id"}
	invalidInputs := &HTTPExampleInputs{Method: "DELETE", URI: "http://localhost:3000/v2/service_instances/INVALID/service_bindings/bar?plan_id=my-plan&service_id=my-service-id"}

	BeforeEach(func() {
		code, _ := brokerClient.ProvisionInstance("foo", "shared")
		Ω(code).Should(Equal(201))
	})

	AfterEach(func() {
		brokerClient.DeprovisionInstance("foo", "shared")
	})

	Context("with valid instance", func() {
		HTTPResponseShouldContainExpectedHTTPStatusCode(validInputs, 200)
		HTTPResponseBodyShouldBeEmptyJSON(validInputs)
	})

	Context("with invalid instance", func() {
		HTTPResponseShouldContainExpectedHTTPStatusCode(invalidInputs, 410)
		HTTPResponseBodyShouldBeEmptyJSON(validInputs)
	})
})
