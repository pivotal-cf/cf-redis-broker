package brokerintegration_test

import (
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"encoding/json"

	"github.com/pivotal-cf/cf-redis-broker/brokerconfig"
	"github.com/pivotal-cf/cf-redis-broker/debug"
	"github.com/pivotal-cf/cf-redis-broker/integration"
	"github.com/pivotal-cf/cf-redis-broker/integration/helpers"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var brokerPort uint = 3000

var brokerSession *gexec.Session
var monitorSession *gexec.Session
var backupExecutablePath string
var brokerConfig brokerconfig.Config
var brokerClient *integration.BrokerClient
var agentRequests []*http.Request
var agentResponseStatus = http.StatusOK

func TestBrokerintegration(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit_brokerintegration.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Broker Integration Suite", []Reporter{junitReporter})
}

var _ = BeforeEach(func() {
	helpers.SafelyResetAllDirectories()
})

var _ = BeforeSuite(func() {
	helpers.SafelyResetAllDirectories()
	brokerConfig = integration.LoadBrokerConfig("broker.yml")
	brokerSession = integration.BuildAndLaunchBroker("broker.yml")

	brokerClient = &integration.BrokerClient{Config: &brokerConfig}

	Ω(helpers.ServiceAvailable(brokerPort)).Should(BeTrue())
	startFakeAgent(&agentRequests, &agentResponseStatus)
})

var _ = AfterSuite(func() {
	helpers.KillProcess(brokerSession)
})

func getRedisProcessCount() int {
	scriptPath, filepathErr := helpers.AssetPath("redis_process_count.sh")
	Ω(filepathErr).NotTo(HaveOccurred())

	output, cmdErr := exec.Command(scriptPath).Output()
	Ω(cmdErr).NotTo(HaveOccurred())

	result, numberParseErr := strconv.Atoi(strings.TrimSpace(string(output)))
	Ω(numberParseErr).NotTo(HaveOccurred())
	return result
}

func getDebugInfo() debug.Info {
	_, bodyBytes := integration.ExecuteAuthenticatedHTTPRequest("GET", "http://localhost:3000/debug", brokerConfig.AuthConfiguration.Username, brokerConfig.AuthConfiguration.Password)
	debugInfo := debug.Info{}

	err := json.Unmarshal(bodyBytes, &debugInfo)
	Ω(err).ShouldNot(HaveOccurred())

	return debugInfo
}
