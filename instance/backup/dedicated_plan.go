package backup

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/pivotal-cf/cf-redis-broker/redisinstance"
)

type dedicatedPlan struct {
	brokerEndpoint string
	username       string
	password       string
}

func NewDedicatedPlan(brokerEndpoint, username, password string) *dedicatedPlan {
	return &dedicatedPlan{
		brokerEndpoint: brokerEndpoint,
		username:       username,
		password:       password,
	}
}

func (p *dedicatedPlan) InstanceID(string, nodeIP string) (string, error) {
	query := url.Values{}
	query.Set("host", nodeIP)
	requestURL := fmt.Sprintf("%s?%s", p.brokerEndpoint, query.Encode())

	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return "", err
	}

	req.SetBasicAuth(p.username, p.password)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Unexpected response code %d from endpoint", res.StatusCode)
	}

	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return "", err
	}

	response := redisinstance.Response{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", err
	}

	return response.InstanceID, nil
}
