package integration

import (
	"encoding/json"
	"fmt"

	"github.com/pivotal-cf/cf-redis-broker/brokerconfig"
)

type BrokerClient struct {
	Config *brokerconfig.Config
}

func (brokerClient *BrokerClient) ProvisionInstance(instanceID string, plan string) (int, []byte) {
	planID, found := map[string]string{
		"shared":    "C210CA06-E7E5-4F5D-A5AA-7A2C51CC290E",
		"dedicated": "74E8984C-5F8C-11E4-86BE-07807B3B2589",
	}[plan]

	if !found {
		panic("invalid plan name:" + plan)
	}

	payload := struct {
		PlanID string `json:"plan_id"`
	}{
		PlanID: planID,
	}

	payloadBytes, err := json.Marshal(&payload)
	if err != nil {
		panic("unable to marshal the payload to provision instance")
	}

	return ExecuteAuthenticatedHTTPRequestWithBody("PUT",
		instanceURI(instanceID, brokerClient.Config.Port),
		brokerClient.Config.AuthConfiguration.Username,
		brokerClient.Config.AuthConfiguration.Password,
		payloadBytes)
}

func instanceURI(instanceID, brokerPort string) string {
	return fmt.Sprintf("http://localhost:%s/v2/service_instances/%s", brokerPort, instanceID)
}
