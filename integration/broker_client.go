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
		brokerClient.InstanceURI(instanceID),
		brokerClient.Config.AuthConfiguration.Username,
		brokerClient.Config.AuthConfiguration.Password,
		payloadBytes)
}

func (brokerClient *BrokerClient) MakeCatalogRequest() (int, []byte) {
	return brokerClient.executeAuthenticatedRequest("GET", "http://localhost:3000/v2/catalog")
}

func (brokerClient *BrokerClient) BindInstance(instanceID, bindingID string) (int, []byte) {
	return brokerClient.executeAuthenticatedRequest("PUT", brokerClient.BindingURI(instanceID, bindingID))
}

func (brokerClient *BrokerClient) UnbindInstance(instanceID, bindingID string) (int, []byte) {
	return brokerClient.executeAuthenticatedRequest("DELETE", brokerClient.BindingURI(instanceID, bindingID))
}

func (brokerClient *BrokerClient) DeprovisionInstance(instanceID string) (int, []byte) {
	return brokerClient.executeAuthenticatedRequest("DELETE", brokerClient.InstanceURI(instanceID))
}

func (brokerClient *BrokerClient) executeAuthenticatedRequest(httpMethod, url string) (int, []byte) {
	return ExecuteAuthenticatedHTTPRequest(httpMethod, url, brokerClient.Config.AuthConfiguration.Username, brokerClient.Config.AuthConfiguration.Password)
}

func (brokerClient *BrokerClient) InstanceURI(instanceID string) string {
	return fmt.Sprintf("http://localhost:%s/v2/service_instances/%s", brokerClient.Config.Port, instanceID)
}

func (brokerClient *BrokerClient) BindingURI(instanceID, bindingID string) string {
	return brokerClient.InstanceURI(instanceID) + "/service_bindings/" + bindingID
}
