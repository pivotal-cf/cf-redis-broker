package backupconfig_test

import (
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"testing"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit_backupconfig.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Backup Config Suite", []Reporter{junitReporter})
}
