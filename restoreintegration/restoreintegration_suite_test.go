package restoreintegration_test

import (
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/cf-redis-broker/integration/helpers"

	"testing"
)

func TestRestore(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit_restoreintegration.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Restore Integration Suite", []Reporter{junitReporter})
}

var restoreExecutablePath string

var _ = BeforeSuite(func() {
	if helpers.ServiceAvailable(6379) {
		panic("something is already using the dedicated redis port!")
	}
	restoreExecutablePath = helpers.BuildExecutable("github.com/pivotal-cf/cf-redis-broker/cmd/restore")
})
