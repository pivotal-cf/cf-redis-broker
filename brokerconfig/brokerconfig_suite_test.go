package brokerconfig_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestBrokerconfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Brokerconfig Suite")
}
