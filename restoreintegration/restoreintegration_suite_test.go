package restoreintegration_test

import (
	"io"
	"log"
	"os"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

func TestRestore(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit_restoreintegration.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Restore Integration Suite", []Reporter{junitReporter})
}

var restoreExecutablePath string

func buildExecutable(sourcePath string) string {
	executable, err := gexec.Build(sourcePath)
	if err != nil {
		log.Fatalf("executable %s could not be built: %s", sourcePath, err)
		os.Exit(1)
	}
	return executable
}

var _ = BeforeSuite(func() {
	restoreExecutablePath = buildExecutable("github.com/pivotal-cf/cf-redis-broker/cmd/restore")
})

func copyFile(sourcePath, destinationPath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(destinationPath)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}
