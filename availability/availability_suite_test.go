package availability_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestAvailability(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Availability Suite")
}
