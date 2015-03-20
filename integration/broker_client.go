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
		brokerClient.instanceURI(instanceID),
		brokerClient.Config.AuthConfiguration.Username,
		brokerClient.Config.AuthConfiguration.Password,
		payloadBytes)
}

func (brokerClient *BrokerClient) BindInstance(instanceID, bindingID string) (int, []byte) {
	return ExecuteAuthenticatedHTTPRequest("PUT", brokerClient.bindingURI(instanceID, bindingID), brokerClient.Config.AuthConfiguration.Username, brokerClient.Config.AuthConfiguration.Password)
}

func (brokerClient *BrokerClient) UnbindInstance(instanceID, bindingID string) (int, []byte) {
	return ExecuteAuthenticatedHTTPRequest("DELETE", brokerClient.bindingURI(instanceID, bindingID), brokerClient.Config.AuthConfiguration.Username, brokerClient.Config.AuthConfiguration.Password)
}

func (brokerClient *BrokerClient) instanceURI(instanceID string) string {
	return fmt.Sprintf("http://localhost:%s/v2/service_instances/%s", brokerClient.Config.Port, instanceID)
}

func (brokerClient *BrokerClient) bindingURI(instanceID, bindingID string) string {
	return brokerClient.instanceURI(instanceID) + "/service_bindings/" + bindingID
}
