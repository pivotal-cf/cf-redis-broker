package credentials_test

import (
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"testing"
)

func TestCredentials(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit_credentials.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Credentials Suite", []Reporter{junitReporter})
}
