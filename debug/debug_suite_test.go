package debug_test

import (
	"net/http"
	"os"
	"path"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/cf-redis-broker/brokerconfig"
	"github.com/pivotal-cf/cf-redis-broker/debug"
	"github.com/pivotal-cf/cf-redis-broker/redis"

	"testing"
)

func TestDebug(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit_debug.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Debug Suite", []Reporter{junitReporter})
}

var dirs []string

var _ = BeforeSuite(func() {
	RemoveAndRecreateDir("/tmp/redis-config-dir")

	dirs = []string{"/tmp/to/redis", "/tmp/redis/data/directory", "/tmp/redis/log/directory"}
	for _, dir := range dirs {
		err := os.MkdirAll(dir, 0755)
		Ω(err).ShouldNot(HaveOccurred())
	}
	_, err := os.Create("/tmp/to/redis/config.conf")
	Ω(err).ShouldNot(HaveOccurred())
	path, err := filepath.Abs(path.Join("..", "brokerconfig", "assets", "test_config.yml"))
	Ω(err).ToNot(HaveOccurred())
	config, err := brokerconfig.ParseConfig(path)
	Ω(err).NotTo(HaveOccurred())

	repo, err := redis.NewRemoteRepository(&redis.RemoteAgentClient{}, config)
	Ω(err).NotTo(HaveOccurred())

	handler := debug.NewHandler(repo)

	http.HandleFunc("/debug", handler)
	go func() {
		defer GinkgoRecover()
		err := http.ListenAndServe("localhost:3000", nil)
		Expect(err).ToNot(HaveOccurred())
	}()
})

var _ = AfterSuite(func() {
	for _, dir := range dirs {
		err := os.RemoveAll(dir)
		Ω(err).ShouldNot(HaveOccurred())
	}
})

func RemoveAndRecreateDir(path string) {
	err := os.RemoveAll(path)
	Ω(err).ShouldNot(HaveOccurred())
	err = os.MkdirAll(path, 0755)
	Ω(err).ShouldNot(HaveOccurred())
}
