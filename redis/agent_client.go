package redis

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

type Credentials struct {
	Port     int    `json:"port"`
	Password string `json:"password"`
}

type AgentClient interface {
	Reset(hostIP string) error
	Credentials(hostIP string) (Credentials, error)
}

type RemoteAgentClient struct {
}

func (agentClient *RemoteAgentClient) Reset(hostIP string) error {
	url := fmt.Sprintf("http://%s:9876/", hostIP)
	request, err := http.NewRequest("DELETE", url, nil)

	if err != nil {
		return err
	}

	response, err := http.DefaultClient.Do(request)

	if err != nil {
		return err
	}

	if response.StatusCode != 200 {
		return errors.New(fmt.Sprintf("Expected status code 200, received %d", response.StatusCode))
	}

	return nil
}

func (agentClient *RemoteAgentClient) Credentials(hostIP string) (Credentials, error) {
	url := fmt.Sprintf("http://%s:9876/", hostIP)
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return Credentials{}, err
	}

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
