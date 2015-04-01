package agentintegration_test

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cloudfoundry-incubator/candiedyaml"
	"github.com/garyburd/redigo/redis"
	"github.com/onsi/gomega/gexec"

	"github.com/pivotal-cf/cf-redis-broker/agentconfig"
	"github.com/pivotal-cf/cf-redis-broker/integration/helpers"
	"github.com/pivotal-cf/cf-redis-broker/redisconf"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"testing"
)

var redisConfPath string
var originalConf redisconf.Conf

func TestAgentintegration(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit_agentintegration.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Agent Integration Suite", []Reporter{junitReporter})
}

func redisNotWritingAof(redisConn redis.Conn) func() bool {
	return func() bool {
		out, _ := redis.String(redisConn.Do("INFO", "persistence"))
		return strings.Contains(out, "aof_pending_rewrite:0") &&
			strings.Contains(out, "aof_rewrite_scheduled:0") &&
			strings.Contains(out, "aof_rewrite_in_progress:0")
	}
}

func startAgentWithConfig(config *agentconfig.Config) *gexec.Session {
	configFile, err := ioutil.TempFile("", "config.yml")
	Expect(err).ToNot(HaveOccurred())

	encoder := candiedyaml.NewEncoder(configFile)
	err = encoder.Encode(config)
	Ω(err).ShouldNot(HaveOccurred())
	configFile.Close()

	agentPath, err := gexec.Build("github.com/pivotal-cf/cf-redis-broker/cmd/agent")
	Ω(err).ShouldNot(HaveOccurred())

	session, err := gexec.Start(
		exec.Command(agentPath, fmt.Sprintf("-agentConfig=%s", configFile.Name())),
		GinkgoWriter,
		GinkgoWriter,
	)
	Ω(err).ShouldNot(HaveOccurred())

	return session
}

func startAgentWithDefaultConfig() *gexec.Session {
	dir, err := ioutil.TempDir("", "redisconf-test")
	redisConfPath = filepath.Join(dir, "redis.conf")

	originalConf = redisconf.New(
		redisconf.Param{Key: "requirepass", Value: "thepassword"},
		redisconf.Param{Key: "port", Value: "6379"},
	)

	err = originalConf.Save(redisConfPath)
	Expect(err).ToNot(HaveOccurred())

	config := &agentconfig.Config{
		DefaultConfPath:     helpers.AssetPath("redis.conf.default"),
		ConfPath:            redisConfPath,
		MonitExecutablePath: "assets/fake_monit",
		Port:                "9876",
		AuthConfiguration: agentconfig.AuthConfiguration{
			Username: "admin",
			Password: "secret",
		},
	}

	session := startAgentWithConfig(config)
	Expect(helpers.ServiceAvailable(9876)).To(BeTrue())
	return session
}

func stopAgent(session *gexec.Session) {
	helpers.KillProcess(session)
	helpers.ResetTestDirs()
}

func startRedis(confPath string) (*gexec.Session, redis.Conn) {
	redisSession, err := gexec.Start(exec.Command("redis-server", confPath), GinkgoWriter, GinkgoWriter)
	Ω(err).ShouldNot(HaveOccurred())

	conf, err := redisconf.Load(confPath)
	Ω(err).ShouldNot(HaveOccurred())

	port, err := strconv.Atoi(conf.Get("port"))
	Ω(err).ShouldNot(HaveOccurred())

	Expect(helpers.ServiceAvailable(uint(port))).To(BeTrue())

	redisConn := helpers.BuildRedisClient(uint(port), "localhost", conf.Get("requirepass"))

	return redisSession, redisConn
}
