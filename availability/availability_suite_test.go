package availability_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestAvailability(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Availability Suite")
}
