package system_test

import (
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"testing"
)

func TestSystem(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit_system.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "System Suite", []Reporter{junitReporter})
}
