package id_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestId(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Instance ID Suite")
}
