package resetter_test

import (
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"testing"
)

func TestResetter(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit_resetter.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Resetter Suite", []Reporter{junitReporter})
}
