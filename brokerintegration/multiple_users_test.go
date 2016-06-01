package brokerintegration_test

import (
	"encoding/json"

	"github.com/pborman/uuid"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Multiple users", func() {

	var instanceIDs []string

	BeforeEach(func() {
		instanceIDs = []string{uuid.NewRandom().String(), uuid.NewRandom().String()}
		for _, instanceID := range instanceIDs {
			statusCode, _ := brokerClient.ProvisionInstance(instanceID, "shared")
			Ω(statusCode).To(Equal(201))
		}
	})

	It("assigns different ports for each instance", func() {
		ports := []uint{}
		for _, instanceID := range instanceIDs {
			_, body := brokerClient.BindInstance(instanceID, "foo")

			var parsedJSON map[string]interface{}
			json.Unmarshal(body, &parsedJSON)

			credentials := parsedJSON["credentials"].(map[string]interface{})

			port := uint(credentials["port"].(float64))
			ports = append(ports, port)
		}
		Ω(ports[0]).ToNot(Equal(ports[1]))
	})

	AfterEach(func() {
		for _, instanceID := range instanceIDs {
			brokerClient.DeprovisionInstance(instanceID)
		}
	})
})
