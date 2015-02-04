package client_test

import (
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"testing"
)

func TestClient(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit_redis_client.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Redis Client Suite", []Reporter{junitReporter})
}
