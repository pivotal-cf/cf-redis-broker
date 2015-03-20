package backupintegration_test

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	redisclient "github.com/garyburd/redigo/redis"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf/cf-redis-broker/availability"
	"github.com/pivotal-cf/cf-redis-broker/brokerconfig"
	"github.com/pivotal-cf/cf-redis-broker/integration"

	"testing"
	"time"
)

var backupExecutablePath string
var brokerConfig brokerconfig.Config
var brokerClient *integration.BrokerClient
var brokerSession *gexec.Session
var brokerPort uint = 3000

func TestBackupintegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Backupintegration Suite")
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

func killProcess(session *gexec.Session) {
	session.Terminate().Wait()
	Eventually(session).Should(gexec.Exit())
}

func buildRedisClient(port uint, host string, password string) redisclient.Conn {
	url := fmt.Sprintf("%s:%d", host, port)

	client, err := redisclient.Dial("tcp", url)
	Ω(err).NotTo(HaveOccurred())

	_, err = client.Do("AUTH", password)
	Ω(err).NotTo(HaveOccurred())

	return client
}
