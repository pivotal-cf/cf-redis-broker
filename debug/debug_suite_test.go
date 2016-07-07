package debug_test

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/cf-redis-broker/brokerconfig"
	"github.com/pivotal-cf/cf-redis-broker/debug"
	"github.com/pivotal-cf/cf-redis-broker/integration/helpers"
	"github.com/pivotal-cf/cf-redis-broker/redis"
	"github.com/pivotal-golang/lager"

	"testing"
)

func TestDebug(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit_debug.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Debug Suite", []Reporter{junitReporter})
}

var dirs []string

var _ = BeforeSuite(func() {
	helpers.ResetTestDirs()

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

	logger := lager.NewLogger("foo")
	repo, err := redis.NewRemoteRepository(&redis.RemoteAgentClient{}, config, logger)
	Ω(err).NotTo(HaveOccurred())

	handler := debug.NewHandler(repo)

	http.HandleFunc("/debug", handler)
	go func() {
		defer GinkgoRecover()
		err := http.ListenAndServe("localhost:3000", nil)
		Expect(err).ToNot(HaveOccurred())
	}()

	client := http.Client{}
	for i := 0; i < 10; i++ {
		_, err = client.Get("http://localhost:3000/debug")
		if err == nil {
			break
		}
		time.Sleep(time.Second * 1)
		if i == 9 {
			Fail("Timed out waiting for debug handler setup")
		}
	}
})

var _ = AfterSuite(func() {
	for _, dir := range dirs {
		err := os.RemoveAll(dir)
		Ω(err).ShouldNot(HaveOccurred())
	}
})
