package brokerintegration_test

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"encoding/json"

	"github.com/cloudfoundry-incubator/candiedyaml"
	"github.com/mitchellh/goamz/s3/s3test"

	redisclient "github.com/garyburd/redigo/redis"
	"github.com/pivotal-cf/cf-redis-broker/availability"
	"github.com/pivotal-cf/cf-redis-broker/brokerconfig"
	"github.com/pivotal-cf/cf-redis-broker/debug"
	"github.com/pivotal-cf/cf-redis-broker/integration"
	"github.com/pivotal-cf/cf-redis-broker/process"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var brokerPort uint = 3000

var brokerSession *gexec.Session
var monitorSession *gexec.Session
var processMonitorPath string
var backupExecutablePath string
var brokerConfig brokerconfig.Config
var previousBackupEndpointUrl string
var fakeAgent *httptest.Server

func TestBrokerintegration(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit_brokerintegration.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Broker Integration Suite", []Reporter{junitReporter})
}

func safelyResetAllDirectories() {
	waitUntilNoRunningRedis(10.0)

	if monitorSession != nil {
		checker := &process.ProcessChecker{}
		Ω(checker.Alive(monitorSession.Command.Process.Pid)).Should(BeFalse())
	}

	removeAndRecreateDir("/tmp/redis-data-dir")
	removeAndRecreateDir("/tmp/redis-log-dir")
	removeAndRecreateDir("/tmp/redis-config-dir")
}

var _ = BeforeEach(func() {
	safelyResetAllDirectories()
})

var _ = AfterEach(func() {
	waitUntilNoRunningRedis(10.0)
})

var _ = BeforeSuite(func() {
	s3TestServerConfig := &s3test.Config{
		Send409Conflict: true,
	}
	s3testServer, err := s3test.NewServer(s3TestServerConfig)
	Ω(err).ToNot(HaveOccurred())

	safelyResetAllDirectories()
	loadBrokerConfig()

	previousBackupEndpointUrl = brokerConfig.RedisConfiguration.BackupConfiguration.EndpointUrl
	brokerConfig.RedisConfiguration.BackupConfiguration.EndpointUrl = s3testServer.URL()
	saveBrokerConfig()

	backupExecutablePath = buildExecutable("github.com/pivotal-cf/cf-redis-broker/cmd/backup")

	brokerSession = buildAndLaunchBroker("broker.yml")

	processMonitorPath = buildExecutable("github.com/pivotal-cf/cf-redis-broker/cmd/processmonitor")

	startFakeAgent()

	ensurePortAvailable(brokerPort)
})

var _ = AfterSuite(func() {
	stopFakeAgent()

	brokerConfig.RedisConfiguration.BackupConfiguration.EndpointUrl = previousBackupEndpointUrl
	saveBrokerConfig()
	killProcess(brokerSession)
})

func startFakeAgent() {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Ω([]string{"DELETE", "GET"}).Should(ContainElement(r.Method))
		Ω(r.URL.Path).Should(Equal("/"))
		w.WriteHeader(http.StatusOK)
		if r.Method == "GET" {
			w.Write([]byte("{\"port\": 12345, \"password\": \"super-secret\"}"))
		}
	})

	fakeAgent = httptest.NewUnstartedServer(handler)
	listener, err := net.Listen("tcp", "127.0.0.1:9876")
	Ω(err).ShouldNot(HaveOccurred())

	fakeAgent.Listener = listener
	fakeAgent.Start()
}

func stopFakeAgent() {
	fakeAgent.Close()
}

func loadBrokerConfig() {
	var err error
	brokerConfig, err = brokerconfig.ParseConfig(brokerConfigPath())
	Ω(err).NotTo(HaveOccurred())
}

func saveBrokerConfig() {
	configFile, err := os.Create(brokerConfigPath())
	Ω(err).ToNot(HaveOccurred())
	encoder := candiedyaml.NewEncoder(configFile)
	err = encoder.Encode(brokerConfig)
	Ω(err).ToNot(HaveOccurred())
}

func brokerConfigPath() string {
	path, err := assetPath("broker.yml")
	Ω(err).ToNot(HaveOccurred())
	return path
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

func relaunchProcessMonitorWithConfig(brokerConfigName string) {
	killProcess(monitorSession)
	monitorSession = launchProcessWithBrokerConfig(processMonitorPath, brokerConfigName)
}

func sendUsr1ToProcessMonitor() {
	monitorSession.Signal(syscall.SIGUSR1)
}

func buildExecutable(sourcePath string) string {
	executable, err := gexec.Build(sourcePath)
	if err != nil {
		log.Fatalf("executable %s could not be built: %s", sourcePath, err)
		os.Exit(1)
	}
	return executable
}

func launchProcessWithBrokerConfig(processPath string, brokerConfigName string) *gexec.Session {
	brokerConfigFile, filePathErr := assetPath(brokerConfigName)
	Ω(filePathErr).ToNot(HaveOccurred())

	os.Setenv("BROKER_CONFIG_PATH", brokerConfigFile)
	processCmd := exec.Command(processPath)
	processCmd.Stdout = GinkgoWriter
	processCmd.Stderr = GinkgoWriter
	return runCommand(processCmd)
}

func switchBroker(config string) {
	killProcess(brokerSession)
	safelyResetAllDirectories()
	brokerSession = buildAndLaunchBroker(config)
	ensurePortAvailable(brokerPort)
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

func waitUntilNoRunningRedis(timeout float64) {
	if timeout < 0 {
		panic("Timed out waiting for redises to shut down")
	}

	processCount := getRedisProcessCount()
	if processCount == 0 {
		return
	}

	time.Sleep(time.Millisecond * 100)
	waitUntilNoRunningRedis(timeout - 0.1)
}

func assetPath(filename string) (string, error) {
	return filepath.Abs(path.Join("assets", filename))
}

func executeHTTPRequest(method string, uri string) (int, []byte) {
	client := &http.Client{}
	req, err := http.NewRequest(method, uri, nil)
	Ω(err).ToNot(HaveOccurred())
	resp, err := client.Do(req)
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	Ω(err).ToNot(HaveOccurred())

	Ω(err).ToNot(HaveOccurred())
	return resp.StatusCode, body
}

func executeAuthenticatedHTTPRequest(method string, uri string) (int, []byte) {
	return integration.ExecuteAuthenticatedHTTPRequest(method, uri, brokerConfig.AuthConfiguration.Username, brokerConfig.AuthConfiguration.Password)
}

func executeAuthenticatedHTTPRequestWithBody(method, uri string, body []byte) (int, []byte) {
	return integration.ExecuteAuthenticatedHTTPRequestWithBody(method, uri, brokerConfig.AuthConfiguration.Username, brokerConfig.AuthConfiguration.Password, body)
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

	return executeAuthenticatedHTTPRequestWithBody("PUT", instanceURI(instanceID), payloadBytes)
}

func bindInstance(instanceID, bindingID string) (int, []byte) {
	return executeAuthenticatedHTTPRequest("PUT", bindingURI(instanceID, bindingID))
}

func unbindInstance(instanceID, bindingID string) (int, []byte) {
	return executeAuthenticatedHTTPRequest("DELETE", bindingURI(instanceID, bindingID))
}

func deprovisionInstance(instanceID string) (int, []byte) {
	return executeAuthenticatedHTTPRequest("DELETE", instanceURI(instanceID))
}

func instanceURI(instanceID string) string {
	return fmt.Sprintf("http://localhost:%d/v2/service_instances/%s", brokerPort, instanceID)
}

func bindingURI(instanceID, bindingID string) string {
	return instanceURI(instanceID) + "/service_bindings/" + bindingID
}

func BuildRedisClient(port uint, host string, password string) redisclient.Conn {
	url := fmt.Sprintf("%s:%d", host, port)

	client, err := redisclient.Dial("tcp", url)
	Ω(err).NotTo(HaveOccurred())

	_, err = client.Do("AUTH", password)
	Ω(err).NotTo(HaveOccurred())

	return client
}

func portAvailableChecker(port uint) func() bool {
	return func() bool {
		return portAvailable(port)
	}
}

func portAvailable(port uint) bool {
	address, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		return false
	}

	err = availability.Check(address, 10*time.Second)
	if err != nil {
		return false
	}

	return true
}

func ensurePortAvailable(port uint) {
	success := portAvailable(port)
	Ω(success).Should(BeTrue())
}

func killRedisProcess(instanceID string) {
	pidFilePath, err := filepath.Abs(path.Join(brokerConfig.RedisConfiguration.InstanceDataDirectory, instanceID, "redis-server.pid"))
	Ω(err).ToNot(HaveOccurred())

	fileContent, err := ioutil.ReadFile(pidFilePath)
	Ω(err).ToNot(HaveOccurred())

	pid, err := strconv.ParseInt(strings.TrimSpace(string(fileContent)), 10, 32)
	Ω(err).ToNot(HaveOccurred())

	process, err := os.FindProcess(int(pid))
	Ω(err).ToNot(HaveOccurred())

	err = process.Kill()
	Ω(err).ToNot(HaveOccurred())

	process.Wait()
}

func getDebugInfo() debug.Info {
	_, bodyBytes := executeAuthenticatedHTTPRequest("GET", "http://localhost:3000/debug")
	debugInfo := debug.Info{}

	err := json.Unmarshal(bodyBytes, &debugInfo)
	Ω(err).ShouldNot(HaveOccurred())

	return debugInfo
}
