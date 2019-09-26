package brokerintegration_test

import (
	"os/exec"
	"strconv"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"github.com/pivotal-cf/cf-redis-broker/brokerconfig"
	"github.com/pivotal-cf/cf-redis-broker/integration"
	"github.com/pivotal-cf/cf-redis-broker/integration/helpers"
)

var (
	brokerSession        *gexec.Session
	monitorSession       *gexec.Session
	brokerExecutablePath string
	brokerConfig         brokerconfig.Config
	brokerClient         *integration.BrokerClient
	brokerPort           uint = 3000
)

func TestBrokerintegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Broker Integration Suite")
}

var _ = BeforeEach(func() {
	helpers.ResetTestDirs()
})

var _ = BeforeSuite(func() {
	helpers.ResetTestDirs()

	brokerExecutablePath = integration.BuildBroker()
	brokerSession = integration.LaunchProcessWithBrokerConfig(brokerExecutablePath, "broker.yml")

	brokerConfig = integration.LoadBrokerConfig("broker.yml")

	brokerClient = &integration.BrokerClient{Config: &brokerConfig}

	Ω(helpers.ServiceAvailable(brokerPort)).Should(BeTrue())
})

var _ = AfterSuite(func() {
	helpers.KillProcess(brokerSession)
})

func getRedisProcessCount() int {
	scriptPath := helpers.AssetPath("redis_process_count.sh")

	output, cmdErr := exec.Command(scriptPath).Output()
	Ω(cmdErr).NotTo(HaveOccurred())

	result, numberParseErr := strconv.Atoi(strings.TrimSpace(string(output)))
	Ω(numberParseErr).NotTo(HaveOccurred())
	return result
}
