package dedicated_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestDedicated(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Dedicated Plan Suite")
}
