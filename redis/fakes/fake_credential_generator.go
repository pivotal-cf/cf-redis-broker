package fakes

type FakeCredentialGenerator struct{}

func (FakeCredentialGenerator) GenerateCredentials() string {
	return "somepassword"
}
