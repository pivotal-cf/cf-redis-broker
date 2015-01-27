package redis

import "github.com/dchest/uniuri"

type CredentialsGenerator interface {
	GenerateCredentials() string
}

type RandomCredentialsGenerator struct{}

func (RandomCredentialsGenerator) GenerateCredentials() string {
	return uniuri.New()
}
