package consistency_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestConsistency(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Consistency Suite")
}
