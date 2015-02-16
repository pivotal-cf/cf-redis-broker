package redis

import (
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

func (agentClient *RemoteAgentClient) Reset(hostIP string) error {
	url := fmt.Sprintf("http://%s:9876/", hostIP)
	request, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	request.SetBasicAuth(agentClient.HttpAuth.Username, agentClient.HttpAuth.Password)

	response, err := http.DefaultClient.Do(request)

	if err != nil {
		return err
	}

	if response.StatusCode != 200 {
		body, _ := ioutil.ReadAll(response.Body)
		return errors.New(fmt.Sprintf("Expected status code 200, received %d, %s", response.StatusCode, string(body)))
	}

	return nil
}

func (agentClient *RemoteAgentClient) Credentials(hostIP string) (Credentials, error) {
	url := fmt.Sprintf("http://%s:9876/", hostIP)
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return Credentials{}, err
	}

	request.SetBasicAuth(agentClient.HttpAuth.Username, agentClient.HttpAuth.Password)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return Credentials{}, err
	}

	if response.StatusCode != http.StatusOK {
		return Credentials{}, errors.New("Received non-200 status code from agent")
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return Credentials{}, err
	}

	credentials := Credentials{}
	err = json.Unmarshal(body, &credentials)
	if err != nil {
		return Credentials{}, err
	}

	return credentials, nil
}
