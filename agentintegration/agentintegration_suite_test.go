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

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	"testing"
)

var redisConfPath string

func TestAgentintegration(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit_agentintegration.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Agent Integration Suite", []Reporter{junitReporter})
}

var _ = BeforeSuite(func() {
	if helpers.ServiceAvailable(6379) {
		panic("something is already using the dedicated redis port!")
	}
	dir, err := ioutil.TempDir("", "redisconf-test")
	Expect(err).ToNot(HaveOccurred())
	redisConfPath = filepath.Join(dir, "redis.conf")
})

func startAgent() *gexec.Session {
	config := &agentconfig.Config{
		DefaultConfPath:     helpers.AssetPath("redis.conf.default"),
		ConfPath:            redisConfPath,
		MonitExecutablePath: helpers.AssetPath("fake_monit"),
		Port:                "9876",
		AuthConfiguration: agentconfig.AuthConfiguration{
			Username: "admin",
			Password: "supersecretpassword",
		},
	}

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

	Expect(helpers.ServiceAvailable(9876)).To(BeTrue())
	return session
}

func stopAgent(session *gexec.Session) {
	helpers.KillProcess(session)
	helpers.ResetTestDirs()
}
