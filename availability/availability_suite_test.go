package availability_test

import (
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"testing"
)

func TestAvailability(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit_availability.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Availability Suite", []Reporter{junitReporter})
}
