package brokerintegration_test

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"encoding/json"

	redisclient "github.com/garyburd/redigo/redis"
	"github.com/pivotal-cf/cf-redis-broker/availability"
	"github.com/pivotal-cf/cf-redis-broker/brokerconfig"
	"github.com/pivotal-cf/cf-redis-broker/debug"
	"github.com/pivotal-cf/cf-redis-broker/integration"

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

func TestBrokerintegration(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit_brokerintegration.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Broker Integration Suite", []Reporter{junitReporter})
}

func safelyResetAllDirectories() {
	removeAndRecreateDir("/tmp/redis-data-dir")
	removeAndRecreateDir("/tmp/redis-log-dir")
	removeAndRecreateDir("/tmp/redis-config-dir")
}

var _ = BeforeEach(func() {
	safelyResetAllDirectories()
})

var _ = BeforeSuite(func() {
	brokerConfig = integration.LoadBrokerConfig("broker.yml")
	brokerSession = integration.BuildAndLaunchBroker("broker.yml")

	brokerClient = &integration.BrokerClient{Config: &brokerConfig}

	backupExecutablePath = buildExecutable("github.com/pivotal-cf/cf-redis-broker/cmd/backup")

	Ω(serviceAvailable(brokerPort)).Should(BeTrue())
})

var _ = AfterSuite(func() {
	killProcess(brokerSession)
})

func removeAndRecreateDir(path string) {
	err := os.RemoveAll(path)
	Ω(err).ShouldNot(HaveOccurred())
	err = os.MkdirAll(path, 0755)
	Ω(err).ShouldNot(HaveOccurred())
}

func buildExecutable(sourcePath string) string {
	executable, err := gexec.Build(sourcePath)
	if err != nil {
		log.Fatalf("executable %s could not be built: %s", sourcePath, err)
		os.Exit(1)
	}
	return executable
}

func killProcess(session *gexec.Session) {
	session.Terminate().Wait()
	Eventually(session).Should(gexec.Exit())
}

func getRedisProcessCount() int {
	scriptPath, filepathErr := assetPath("redis_process_count.sh")
	Ω(filepathErr).NotTo(HaveOccurred())

	output, cmdErr := exec.Command(scriptPath).Output()
	Ω(cmdErr).NotTo(HaveOccurred())

	result, numberParseErr := strconv.Atoi(strings.TrimSpace(string(output)))
	Ω(numberParseErr).NotTo(HaveOccurred())
	return result
}

func assetPath(filename string) (string, error) {
	return filepath.Abs(path.Join("assets", filename))
}

func buildRedisClient(port uint, host string, password string) redisclient.Conn {
	url := fmt.Sprintf("%s:%d", host, port)

	client, err := redisclient.Dial("tcp", url)
	Ω(err).NotTo(HaveOccurred())

	_, err = client.Do("AUTH", password)
	Ω(err).NotTo(HaveOccurred())

	return client
}

func serviceAvailable(port uint) bool {
	address, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		return false
	}

	if err = availability.Check(address, 10*time.Second); err != nil {
		return false
	}

	return true
}

func getDebugInfo() debug.Info {
	_, bodyBytes := integration.ExecuteAuthenticatedHTTPRequest("GET", "http://localhost:3000/debug", brokerConfig.AuthConfiguration.Username, brokerConfig.AuthConfiguration.Password)
	debugInfo := debug.Info{}

	err := json.Unmarshal(bodyBytes, &debugInfo)
	Ω(err).ShouldNot(HaveOccurred())

	return debugInfo
}
