package resetter

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestResetter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Resetter Suite")
}

func doReturns(returns []interfaceAndErr) *do {
	return &do{returnIndex: -1, returns: returns}
}

type interfaceAndErr struct {
	inter interface{}
	err   error
}

type do struct {
	returns     []interfaceAndErr
	returnIndex int
}

func (d *do) sequentially(string, ...interface{}) (interface{}, error) {
	d.returnIndex++
	return d.returns[d.returnIndex].inter, d.returns[d.returnIndex].err
}
