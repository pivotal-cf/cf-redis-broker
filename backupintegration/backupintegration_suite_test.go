package backupintegration_test

import (
	"net"
	"net/http"
	"net/http/httptest"

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
var agentRequests []*http.Request
var agentResponseStatus = http.StatusOK

func TestBackupintegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Backupintegration Suite")
}

var _ = BeforeEach(func() {
	helpers.ResetTestDirs()
})

var _ = BeforeSuite(func() {
	helpers.ResetTestDirs()
	brokerConfig = integration.LoadBrokerConfig("broker.yml")
	brokerSession = integration.LaunchProcessWithBrokerConfig(integration.BuildBroker(), "broker.yml")

	brokerClient = &integration.BrokerClient{Config: &brokerConfig}

	backupExecutablePath = helpers.BuildExecutable("github.com/pivotal-cf/cf-redis-broker/cmd/backup")

	Ω(helpers.ServiceAvailable(brokerPort)).Should(BeTrue())
	startFakeAgent(&agentRequests, &agentResponseStatus)
})

var _ = AfterSuite(func() {
	helpers.KillProcess(brokerSession)
})

func startFakeAgent(agentRequests *[]*http.Request, agentResponseStatus *int) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*agentRequests = append(*agentRequests, r)

		if *agentResponseStatus != http.StatusOK {
			http.Error(w, "", *agentResponseStatus)
			return
		}

		w.WriteHeader(*agentResponseStatus)

		if r.Method == "GET" {
			w.Write([]byte("{\"port\": 6480, \"password\": \"super-secret\"}"))
		}
	})

	listener, err := net.Listen("tcp", ":9876")
	if err != nil {
		panic(err)
	}

	fakeAgent := httptest.NewUnstartedServer(handler)
	fakeAgent.Listener = listener
	fakeAgent.StartTLS()
}
