package configmigratorintegration_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestConfigmigratorintegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Configmigratorintegration Suite")
}
