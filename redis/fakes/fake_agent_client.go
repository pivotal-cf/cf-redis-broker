package fakes

import "github.com/pivotal-cf/cf-redis-broker/redis"

type FakeAgentClient struct {
	ResetHostIPs    []string
	CredentialsFunc func(string) (redis.Credentials, error)

	ResetHandler func(string) error
}

func (fakeAgentClient *FakeAgentClient) Reset(hostIP string) error {
	if fakeAgentClient.ResetHandler == nil {
		fakeAgentClient.ResetHostIPs = append(fakeAgentClient.ResetHostIPs, hostIP)
		return nil
	} else {
		return fakeAgentClient.ResetHandler(hostIP)
	}
}

func (fakeAgentClient *FakeAgentClient) Credentials(hostIp string) (redis.Credentials, error) {
	return fakeAgentClient.CredentialsFunc(hostIp)
}
