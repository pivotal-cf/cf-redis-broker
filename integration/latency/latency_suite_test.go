package latency_test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf/cf-redis-broker/integration"
	"github.com/pivotal-cf/cf-redis-broker/integration/helpers"

	"testing"
)

var (
	latencyExecutablePath string
	redisRunner           *integration.RedisRunner
	latencyDir            string
	latencyFilePath       string
	redisConfigFilePath   string
	latencyConfigFilePath string
	latencyInterval       string
)

func TestLatency(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Latency Suite")
}

type RedisTemplateData struct {
	RedisPort     int
	RedisPassword string
}

type LatencyTemplateData struct {
	LatencyInterval string
	LatencyFilePath string
}

var _ = BeforeSuite(func() {
	var err error
	latencyDir, err = ioutil.TempDir("", "redis-latency-")
	Expect(err).ToNot(HaveOccurred())
	latencyFilePath = filepath.Join(latencyDir, "latency")
	latencyExecutablePath = helpers.BuildExecutable("github.com/pivotal-cf/cf-redis-broker/cmd/latency")

	redisTemplateData := &RedisTemplateData{
		RedisPort: integration.RedisPort,
	}
	redisConfigFilePath = filepath.Join(latencyDir, "redis.conf")
	err = helpers.HandleTemplate(
		helpers.AssetPath("redis.conf.template"),
		redisConfigFilePath,
		redisTemplateData,
	)
	Expect(err).ToNot(HaveOccurred())

	latencyInterval = "1s"
	latencyConfigFilePath = filepath.Join(latencyDir, "latency.yml")

	redisRunner = new(integration.RedisRunner)
	redisRunner.Start([]string{redisConfigFilePath})
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()

	redisRunner.Stop()

	os.RemoveAll(latencyDir)
})
