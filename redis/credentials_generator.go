package redis

import "code.google.com/p/go-uuid/uuid"

type CredentialsGenerator interface {
	GenerateCredentials() string
}

type RandomCredentialsGenerator struct{}

func (RandomCredentialsGenerator) GenerateCredentials() string {
	return uuid.NewRandom().String()
}
