package broker_test

import (
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"testing"
)

func TestBroker(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit_broker.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Broker Suite", []Reporter{junitReporter})
}
