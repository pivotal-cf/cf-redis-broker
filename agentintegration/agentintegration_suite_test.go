package agentintegration_test

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry-incubator/candiedyaml"
	"github.com/garyburd/redigo/redis"
	"github.com/onsi/gomega/gexec"

	"github.com/pivotal-cf/cf-redis-broker/agentconfig"
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

func startAgentWithFile(configPath string) *gexec.Session {
	agentPath, err := gexec.Build("github.com/pivotal-cf/cf-redis-broker/cmd/agent")
	Ω(err).ShouldNot(HaveOccurred())

	session, err := gexec.Start(exec.Command(agentPath, fmt.Sprintf("-agentConfig=%s", configPath)), GinkgoWriter, GinkgoWriter)
	Ω(err).ShouldNot(HaveOccurred())

	return session
}

func startAgentWithConfig(config *agentconfig.Config) *gexec.Session {
	file, err := ioutil.TempFile("", "config.yml")
	Expect(err).ToNot(HaveOccurred())

	encoder := candiedyaml.NewEncoder(file)
	err = encoder.Encode(config)
	file.Close()

	return startAgentWithFile(file.Name())
}

func defaultRedisConfigPath() string {
	defaultConfPath, err := filepath.Abs(path.Join("assets", "redis.conf.default"))
	Ω(err).ShouldNot(HaveOccurred())
	return defaultConfPath
}

func createDefaultRedisConfig() {
	dir, err := ioutil.TempDir("", "redisconf-test")
	redisConfPath = filepath.Join(dir, "redis.conf")

	originalConf = redisconf.New(
		redisconf.Param{Key: "requirepass", Value: "thepassword"},
		redisconf.Param{Key: "port", Value: "6379"},
	)

	err = originalConf.Save(redisConfPath)
	Expect(err).ToNot(HaveOccurred())
}

func startAgentWithDefaultConfig() *gexec.Session {
	createDefaultRedisConfig()

	config := &agentconfig.Config{
		DefaultConfPath:     defaultRedisConfigPath(),
		ConfPath:            redisConfPath,
		MonitExecutablePath: "assets/fake_monit",
		Port:                "9876",
		AuthConfiguration: agentconfig.AuthConfiguration{
			Username: "admin",
			Password: "secret",
		},
	}

	session := startAgentWithConfig(config)
	Eventually(listening("localhost:9876")).Should(BeTrue())
	return session
}

func stopAgent(session *gexec.Session) {
	session.Terminate().Wait()
	Eventually(session).Should(gexec.Exit())

	err := os.Remove(redisConfPath)
	Expect(err).ToNot(HaveOccurred())
}

func buildRedisConn(conf redisconf.Conf) (redis.Conn, error) {
	password := conf.Get("requirepass")
	port := conf.Get("port")
	uri := fmt.Sprintf("127.0.0.1:%s", port)

	redisConn, err := redis.Dial("tcp", uri)
	if err != nil {
		return nil, err
	}

	_, err = redisConn.Do("AUTH", password)
	if err != nil {
		return nil, err
	}

	return redisConn, nil
}

func startRedis(confPath string) (*gexec.Session, redis.Conn) {
	redisSession, err := gexec.Start(exec.Command("redis-server", confPath), GinkgoWriter, GinkgoWriter)
	Ω(err).ShouldNot(HaveOccurred())

	conf, err := redisconf.Load(confPath)
	Ω(err).ShouldNot(HaveOccurred())

	port := conf.Get("port")
	uri := fmt.Sprintf("127.0.0.1:%s", port)

	Eventually(func() bool {
		conn, err := net.Dial("tcp", uri)
		if err == nil {
			conn.Close()
			return true
		}
		return false
	}).Should(BeTrue())

	redisConn, err := buildRedisConn(conf)
	Ω(err).ShouldNot(HaveOccurred())

	return redisSession, redisConn
}

func fileExistsChecker(path string) func() bool {
	return func() bool {
		return fileExists(path)
	}
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func listening(uri string) func() bool {
	return func() bool {
		address, err := net.ResolveTCPAddr("tcp", uri)
		Expect(err).ToNot(HaveOccurred())

		_, err = net.DialTCP("tcp", nil, address)
		return err == nil
	}
}
