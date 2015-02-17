package fakes

import "github.com/pivotal-cf/cf-redis-broker/redis"

type FakeAgentClient struct {
	ResetURLs       []string
	CredentialsFunc func(string) (redis.Credentials, error)

	ResetHandler func(string) error
}

func (fakeAgentClient *FakeAgentClient) Reset(rootURL string) error {
	if fakeAgentClient.ResetHandler == nil {
		fakeAgentClient.ResetURLs = append(fakeAgentClient.ResetURLs, rootURL)
		return nil
	} else {
		return fakeAgentClient.ResetHandler(rootURL)
	}
}

func (fakeAgentClient *FakeAgentClient) Credentials(rootURL string) (redis.Credentials, error) {
	return fakeAgentClient.CredentialsFunc(rootURL)
}
