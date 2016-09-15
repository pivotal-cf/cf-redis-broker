package redis

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/pivotal-cf/cf-redis-broker/agentapi"
)

type Credentials struct {
	Port     int    `json:"port"`
	Password string `json:"password"`
}

type RemoteAgentClient struct {
	protocol string
	port     string
	username string
	password string
}

func NewRemoteAgentClient(port, username, password string, secure bool) *RemoteAgentClient {
	proto := "https"
	if !secure {
		proto = "http"
	}

	return &RemoteAgentClient{
		protocol: proto,
		port:     port,
		username: username,
		password: password,
	}
}

func (client *RemoteAgentClient) Reset(host string) error {
	response, err := client.doAuthenticatedRequest(host, "DELETE", "/")
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusOK {
		return client.agentError(response)
	}

	return nil
}

func (client *RemoteAgentClient) Credentials(host string) (Credentials, error) {
	credentials := Credentials{}

	response, err := client.doAuthenticatedRequest(host, "GET", "/")
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

func (client *RemoteAgentClient) Keycount(host string) (int, error) {
	response, err := client.doAuthenticatedRequest(host, "GET", "/keycount")
	if err != nil {
		return 0, err
	}

	if response.StatusCode != http.StatusOK {
		return 0, client.agentError(response)
	}

	result := new(agentapi.KeycountResponse)
	if err := json.NewDecoder(response.Body).Decode(result); err != nil {
		return 0, err
	}

	return result.Keycount, nil
}

func (client *RemoteAgentClient) agentError(response *http.Response) error {
	body, _ := ioutil.ReadAll(response.Body)
	formattedBody := ""
	if len(body) > 0 {
		formattedBody = fmt.Sprintf(", %s", string(body))
	}
	return errors.New(fmt.Sprintf("Agent error: %d%s", response.StatusCode, formattedBody))
}

func (client *RemoteAgentClient) doAuthenticatedRequest(host, method, path string) (*http.Response, error) {
	url := fmt.Sprintf("%s://%s:%s%s", client.protocol, host, client.port, path)
	request, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	request.SetBasicAuth(client.username, client.password)

	httpClient := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	return httpClient.Do(request)
}
