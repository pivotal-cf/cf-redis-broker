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
	loadBrokerConfig()

	backupExecutablePath = buildExecutable("github.com/pivotal-cf/cf-redis-broker/cmd/backup")

	brokerSession = buildAndLaunchBroker("broker.yml")

	Ω(serviceAvailable(brokerPort)).Should(BeTrue())
})

var _ = AfterSuite(func() {
	killProcess(brokerSession)
})

func loadBrokerConfig() {
	brokerConfigPath, err := assetPath("broker.yml")
	Ω(err).ToNot(HaveOccurred())

	brokerConfig, err = brokerconfig.ParseConfig(brokerConfigPath)
	Ω(err).NotTo(HaveOccurred())
}

func buildAndLaunchBroker(brokerConfigName string) *gexec.Session {
	brokerPath := buildExecutable("github.com/pivotal-cf/cf-redis-broker/cmd/broker")
	return launchProcessWithBrokerConfig(brokerPath, brokerConfigName)
}

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

func launchProcessWithBrokerConfig(executablePath string, brokerConfigName string) *gexec.Session {
	brokerConfigFile, filePathErr := assetPath(brokerConfigName)
	Ω(filePathErr).ToNot(HaveOccurred())

	os.Setenv("BROKER_CONFIG_PATH", brokerConfigFile)
	processCmd := exec.Command(executablePath)
	processCmd.Stdout = GinkgoWriter
	processCmd.Stderr = GinkgoWriter
	return runCommand(processCmd)
}

func killProcess(session *gexec.Session) {
	session.Terminate().Wait()
	Eventually(session).Should(gexec.Exit())
}

func runCommand(cmd *exec.Cmd) *gexec.Session {
	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Ω(err).NotTo(HaveOccurred())
	return session
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

func makeCatalogRequest() (int, []byte) {
	return integration.ExecuteAuthenticatedHTTPRequest("GET", "http://localhost:3000/v2/catalog", brokerConfig.AuthConfiguration.Username, brokerConfig.AuthConfiguration.Password)
}

func provisionInstance(instanceID string, plan string) (int, []byte) {
	planID, found := map[string]string{
		"shared":    "C210CA06-E7E5-4F5D-A5AA-7A2C51CC290E",
		"dedicated": "74E8984C-5F8C-11E4-86BE-07807B3B2589",
	}[plan]

	Expect(found).To(BeTrue())

	payload := struct {
		PlanID string `json:"plan_id"`
	}{
		PlanID: planID,
	}

	payloadBytes, err := json.Marshal(&payload)
	Expect(err).ToNot(HaveOccurred())

	return integration.ExecuteAuthenticatedHTTPRequestWithBody("PUT",
		instanceURI(instanceID),
		brokerConfig.AuthConfiguration.Username,
		brokerConfig.AuthConfiguration.Password,
		payloadBytes)
}

func bindInstance(instanceID, bindingID string) (int, []byte) {
	return integration.ExecuteAuthenticatedHTTPRequest("PUT", bindingURI(instanceID, bindingID), brokerConfig.AuthConfiguration.Username, brokerConfig.AuthConfiguration.Password)
}

func unbindInstance(instanceID, bindingID string) (int, []byte) {
	return integration.ExecuteAuthenticatedHTTPRequest("DELETE", bindingURI(instanceID, bindingID), brokerConfig.AuthConfiguration.Username, brokerConfig.AuthConfiguration.Password)
}

func deprovisionInstance(instanceID string) (int, []byte) {
	return integration.ExecuteAuthenticatedHTTPRequest("DELETE", instanceURI(instanceID), brokerConfig.AuthConfiguration.Username, brokerConfig.AuthConfiguration.Password)
}

func instanceURI(instanceID string) string {
	return fmt.Sprintf("http://localhost:%d/v2/service_instances/%s", brokerPort, instanceID)
}

func bindingURI(instanceID, bindingID string) string {
	return instanceURI(instanceID) + "/service_bindings/" + bindingID
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
