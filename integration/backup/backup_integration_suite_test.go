package backup_integration_test

import (
	"fmt"
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
)

func TestBackupintegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Backup Integration Suite")
}

var _ = BeforeSuite(func() {
	backupExecutablePath = helpers.BuildExecutable("github.com/pivotal-cf/cf-redis-broker/cmd/snapshot")

	redisHost := "127.0.0.1"
	redisPort := integration.RedisPort

	redisRunner = &integration.RedisRunner{}
	redisRunner.Start([]string{"--bind", redisHost, "--port", fmt.Sprintf("%d", redisPort)})
})

var _ = AfterSuite(func() {
	redisRunner.Stop()
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
