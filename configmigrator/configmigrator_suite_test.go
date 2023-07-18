package configmigrator_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

func TestConfigmigrator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Configmigrator Suite")
}
