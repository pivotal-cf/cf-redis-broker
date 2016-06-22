package restoreconfig_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestRestoreconfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Restore Config Suite")
}
