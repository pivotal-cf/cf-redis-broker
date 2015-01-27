package redis_test

import (
	"github.com/pivotal-cf/cf-redis-broker/redis"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("random credentials generator", func() {

	It("generates a random string as password", func() {
		generator := redis.RandomCredentialsGenerator{}
		password1 := generator.GenerateCredentials()
		Ω(password1).NotTo(BeEmpty())
		password2 := generator.GenerateCredentials()
		Ω(password2).NotTo(Equal(password1))
	})
})
