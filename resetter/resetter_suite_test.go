package resetter

import (
	"io/ioutil"
	"os"
	"strings"
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

func interfacesToString(inters []interface{}) (str string) {
	for _, inter := range inters {
		str = str + inter.(string)
	}
	return
}

func makeTempDir() string {
	dir, err := ioutil.TempDir("", "")
	Expect(err).NotTo(HaveOccurred())
	return dir
}

func removeAllIfTemp(dir string) {
	if strings.HasPrefix(dir, os.TempDir()) {
		os.RemoveAll(dir)
		Expect(dir).NotTo(BeAnExistingFile())
	}
}
