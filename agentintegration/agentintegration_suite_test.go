package agentintegration_test

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"

	"github.com/cloudfoundry-incubator/candiedyaml"
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
