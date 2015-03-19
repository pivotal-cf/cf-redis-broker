package configmigrator_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestConfigmigrator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Configmigrator Suite")
}
