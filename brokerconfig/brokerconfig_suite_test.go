package brokerconfig_test

import (
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"testing"
)

func TestBrokerconfig(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit_brokerconfig.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Brokerconfig Suite", []Reporter{junitReporter})
}
