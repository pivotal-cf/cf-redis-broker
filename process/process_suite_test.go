package process_test

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/reporters"
	. "github.com/onsi/gomega"

	"testing"
)

func TestProcess(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit_process.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Process Suite", []Reporter{junitReporter})
}
