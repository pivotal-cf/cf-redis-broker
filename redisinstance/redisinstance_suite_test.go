package redisinstance_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestRedisinstance(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Redisinstance Suite")
}
