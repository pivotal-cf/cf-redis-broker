package resetter_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestResetter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Resetter Suite")
}
