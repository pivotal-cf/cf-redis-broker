package redis

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pivotal-cf/cf-redis-broker/brokerconfig"
)

type Credentials struct {
	Port     int    `json:"port"`
	Password string `json:"password"`
}

type RemoteAgentClient struct {
	HttpAuth brokerconfig.AuthConfiguration
}

func (client *RemoteAgentClient) Reset(rootURL string) error {
	response, err := client.doAuthenticatedRequest(rootURL, "DELETE")
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusOK {
		return client.agentError(response)
	}

	return nil
}

func (client *RemoteAgentClient) Credentials(rootURL string) (Credentials, error) {
	credentials := Credentials{}

	response, err := client.doAuthenticatedRequest(rootURL, "GET")
	if err != nil {
		return credentials, err
	}

	if response.StatusCode != http.StatusOK {
		return credentials, client.agentError(response)
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return credentials, err
	}

	err = json.Unmarshal(body, &credentials)
	if err != nil {
		return credentials, err
	}

	return credentials, nil
}

func (client *RemoteAgentClient) agentError(response *http.Response) error {
	body, _ := ioutil.ReadAll(response.Body)
	formattedBody := ""
	if len(body) > 0 {
		formattedBody = fmt.Sprintf(", %s", string(body))
	}
	return errors.New(fmt.Sprintf("Agent error: %d%s", response.StatusCode, formattedBody))
}

func (client *RemoteAgentClient) doAuthenticatedRequest(rootURL, method string) (*http.Response, error) {
	request, err := http.NewRequest(method, rootURL, nil)
	if err != nil {
		return nil, err
	}

	request.SetBasicAuth(client.HttpAuth.Username, client.HttpAuth.Password)

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	return httpClient.Do(request)
}
