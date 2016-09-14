package fakes

import "github.com/pivotal-cf/cf-redis-broker/redis"

type FakeAgentClient struct {
	ResetHosts      []string
	CredentialsFunc func(string) (redis.Credentials, error)

	ResetHandler func(string) error
}

func (fakeAgentClient *FakeAgentClient) Reset(host string) error {
	if fakeAgentClient.ResetHandler == nil {
		fakeAgentClient.ResetHosts = append(fakeAgentClient.ResetHosts, host)
		return nil
	} else {
		return fakeAgentClient.ResetHandler(host)
	}
}

func (fakeAgentClient *FakeAgentClient) Credentials(host string) (redis.Credentials, error) {
	return fakeAgentClient.CredentialsFunc(host)
}
