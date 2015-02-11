package agentconfig_test

import (
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"testing"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit_agentconfig.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Agent Config Suite", []Reporter{junitReporter})
}
