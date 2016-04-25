package id

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/pivotal-cf/cf-redis-broker/redisinstance"
	"github.com/pivotal-golang/lager"
)

type dedicatedInstanceIDLocator struct {
	brokerEndpoint string
	username       string
	password       string
	logger         lager.Logger
}

func DedicatedInstanceIDLocator(brokerEndpoint, username, password string, logger lager.Logger) InstanceIDLocator {
	return &dedicatedInstanceIDLocator{
		brokerEndpoint: brokerEndpoint,
		username:       username,
		password:       password,
		logger:         logger,
	}
}

func (p *dedicatedInstanceIDLocator) LocateID(string, nodeIP string) (string, error) {
	p.logger.Info(
		"dedicated-instance-id",
		lager.Data{
			"event":   "starting",
			"node_ip": nodeIP,
		},
	)

	query := url.Values{}
	query.Set("host", nodeIP)
	requestURL := fmt.Sprintf("%s?%s", p.brokerEndpoint, query.Encode())

	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return "", err
	}

	req.SetBasicAuth(p.username, p.password)

	p.logger.Info(
		"broker-request",
		lager.Data{
			"event": "starting",
			"url":   requestURL,
		},
	)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		p.logger.Error("broker-request", err, lager.Data{
			"event": "failed",
			"url":   requestURL,
		})
		return "", err
	}

	p.logger.Info(
		"broker-request",
		lager.Data{
			"event": "done",
			"url":   requestURL,
		},
	)

	if res.StatusCode != http.StatusOK {
		var errorMessage error

		if res.StatusCode == 404 {
			errorMessage = fmt.Errorf(
				"Unexpected response code %d from endpoint - check your Redis instance has been provisioned successfully",
				res.StatusCode,
			)
		} else {
			errorMessage = fmt.Errorf("Unexpected response code %d from endpoint", res.StatusCode)
		}

		p.logger.Error("check-response-status", errorMessage, lager.Data{
			"event":         "failed",
			"url":           requestURL,
			"response_code": res.StatusCode,
		})
		return "", errorMessage
	}

	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		p.logger.Error("open-response-body", err, lager.Data{
			"event": "failed",
		})
		return "", err
	}

	response := redisinstance.Response{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		p.logger.Error("unmarshal-response-body", err, lager.Data{
			"event": "failed",
			"body":  body,
		})
		return "", err
	}

	p.logger.Info(
		"dedicated-instance-id",
		lager.Data{
			"event":       "done",
			"node_ip":     nodeIP,
			"instance_id": response.InstanceID,
		},
	)

	return response.InstanceID, nil
}
