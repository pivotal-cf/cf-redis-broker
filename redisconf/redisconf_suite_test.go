package redisconf_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestRedisconf(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Redisconf Suite")
}
