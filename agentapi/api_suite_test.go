package agentapi_test

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"testing"
)

func TestApi(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit_api.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Agent API Suite", []Reporter{junitReporter})
}

func readAll(reader io.Reader) []byte {
	contents, err := ioutil.ReadAll(reader)
	Expect(err).NotTo(HaveOccurred())
	return contents
}

func getAbsPath(path string) string {
	absPath, err := filepath.Abs(path)
	Expect(err).NotTo(HaveOccurred())
	return absPath
}

func unmarshalJSON(rawJSON []byte) map[string]interface{} {
	unmarshaled := make(map[string]interface{})
	err := json.Unmarshal(rawJSON, &unmarshaled)
	Expect(err).NotTo(HaveOccurred())
	return unmarshaled
}
