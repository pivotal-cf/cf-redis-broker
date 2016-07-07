package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pivotal-cf/cf-redis-broker/brokerconfig"
)

var xipIOBackoff []int

func init() {
	xipIOBackoff = []int{10, 30, 30, 60, 0}
}

type BrokerClient struct {
	Config *brokerconfig.Config
}

func (brokerClient *BrokerClient) ProvisionInstance(instanceID string, plan string) (int, []byte) {
	var status int
	var response []byte

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

	for _, i := range xipIOBackoff {
		status, response = ExecuteAuthenticatedHTTPRequestWithBody("PUT",
			brokerClient.InstanceURI(instanceID),
			brokerClient.Config.AuthConfiguration.Username,
			brokerClient.Config.AuthConfiguration.Password,
			payloadBytes,
		)

		if status == http.StatusCreated {
			break // Pass
		}

		if isNotXIPIOHostErr(response) {
			break // Fail
		}

		if i != 0 {
			fmt.Printf("xip.io unavailable; retrying provision in %d seconds\n", i)
			time.Sleep(time.Second * time.Duration(i))
		}
	}

	// TODO - #122030819
	// Currently, the broker's Provision of a Redis instance does not wait until
	// the instance is ready (this seems to be when the log for Redis reports
	// "server started" and Redis' PID file is populated). This artifical wait
	// is a dumb work around until this is fixed.
	time.Sleep(time.Second * 2)

	return status, response
}

func (brokerClient *BrokerClient) MakeCatalogRequest() (int, []byte) {
	return brokerClient.executeAuthenticatedRequest("GET", "http://localhost:3000/v2/catalog")
}

func (brokerClient *BrokerClient) BindInstance(instanceID, bindingID string) (int, []byte) {
	var status int
	var response []byte

	for _, i := range xipIOBackoff {
		status, response = brokerClient.executeAuthenticatedRequest("PUT", brokerClient.BindingURI(instanceID, bindingID))

		if status == http.StatusOK {
			break // Pass
		}

		if isNotXIPIOHostErr(response) {
			break // Fail
		}

		if i != 0 {
			fmt.Printf("xip.io unavailable; retrying bind in %d seconds\n", i)
			time.Sleep(time.Second * time.Duration(i))
		}
	}

	return status, response
}

func (brokerClient *BrokerClient) UnbindInstance(instanceID, bindingID string) (int, []byte) {
	var status int
	var response []byte

	for _, i := range xipIOBackoff {
		status, response = brokerClient.executeAuthenticatedRequest("DELETE", brokerClient.BindingURI(instanceID, bindingID))

		if status == http.StatusOK {
			break // Pass
		}

		if isNotXIPIOHostErr(response) {
			break // Fail
		}

		if i != 0 {
			fmt.Printf("xip.io unavailable; retrying unbind in %d seconds\n", i)
			time.Sleep(time.Second * time.Duration(i))
		}
	}

	return status, response
}

func (brokerClient *BrokerClient) DeprovisionInstance(instanceID string) (int, []byte) {
	var status int
	var response []byte

	for _, i := range xipIOBackoff {
		status, response = brokerClient.executeAuthenticatedRequest("DELETE", brokerClient.InstanceURI(instanceID))

		if status == http.StatusOK {
			break // Pass
		}

		if isNotXIPIOHostErr(response) {
			break // Fail
		}

		if i != 0 {
			fmt.Printf("xip.io unavailable; retrying deprovision in %d seconds\n", i)
			time.Sleep(time.Second * time.Duration(i))
		}
	}

	return status, response
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

func (brokerClient *BrokerClient) InstanceIDFromHost(host string) (int, []byte) {
	return brokerClient.executeAuthenticatedRequest("GET", brokerClient.instanceIDFromHostURI(host))
}

func (brokerClient *BrokerClient) instanceIDFromHostURI(host string) string {
	return fmt.Sprintf("http://localhost:3000/instance?host=%s", host)
}

func isNotXIPIOHostErr(response []byte) bool {
	if !bytes.Contains(response, []byte("no such host")) {
		return true
	}

	return !bytes.Contains(response, []byte("xip.io"))
}
