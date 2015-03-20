package backupintegration_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf/cf-redis-broker/brokerconfig"
	"github.com/pivotal-cf/cf-redis-broker/integration"
	"github.com/pivotal-cf/cf-redis-broker/integration/helpers"

	"testing"
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

var _ = BeforeEach(func() {
	helpers.SafelyResetAllDirectories()
})

var _ = BeforeSuite(func() {
	brokerConfig = integration.LoadBrokerConfig("broker.yml")
	brokerSession = integration.BuildAndLaunchBroker("broker.yml")

	brokerClient = &integration.BrokerClient{Config: &brokerConfig}

	backupExecutablePath = helpers.BuildExecutable("github.com/pivotal-cf/cf-redis-broker/cmd/backup")

	Ω(helpers.ServiceAvailable(brokerPort)).Should(BeTrue())
})

var _ = AfterSuite(func() {
	helpers.KillProcess(brokerSession)
})
