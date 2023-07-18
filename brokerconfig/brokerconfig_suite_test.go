package brokerconfig_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestBrokerconfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Broker Config Suite")
}
