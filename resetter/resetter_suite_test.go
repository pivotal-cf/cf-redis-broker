package resetter_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestResetter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Resetter Suite")
}
