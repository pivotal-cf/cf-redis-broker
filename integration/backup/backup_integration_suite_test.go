package backup_integration_test

import (
	"fmt"
	"net/http"
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf/cf-redis-broker/integration"
	"github.com/pivotal-cf/cf-redis-broker/integration/helpers"

	"testing"
)

var (
	backupExecutablePath string
	awsCliPath           = "aws"
	redisRunner          *integration.RedisRunner
	brokerHost           = "127.0.0.1"
	brokerPort           = 8080
)

func TestBackupintegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Backup Integration Suite")
}

var _ = BeforeSuite(func() {
	backupExecutablePath = helpers.BuildExecutable("github.com/pivotal-cf/cf-redis-broker/cmd/backup")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, "{\"instance_id\": \"this_is_an_instance_id\"}")
	})

	go func() {
		http.ListenAndServe(fmt.Sprintf(":%v", brokerPort), nil)
	}()
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})

func runBackup(configPath string) int {
	cmd := exec.Command(backupExecutablePath, "-config", configPath)
	cmd.Stdout = GinkgoWriter
	cmd.Stderr = GinkgoWriter

	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).ToNot(HaveOccurred())
	Eventually(session, 10).Should(gexec.Exit())

	fmt.Println(string(session.Err.Contents()))

	return session.ExitCode()
}
