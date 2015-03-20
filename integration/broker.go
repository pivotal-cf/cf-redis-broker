package integration

import (
	"log"
	"os"
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf/cf-redis-broker/brokerconfig"
	"github.com/pivotal-cf/cf-redis-broker/integration/helpers"
)

func LoadBrokerConfig(brokerFilename string) brokerconfig.Config {
	brokerConfigPath, err := helpers.AssetPath(brokerFilename)
	Ω(err).ToNot(HaveOccurred())

	brokerConfig, err := brokerconfig.ParseConfig(brokerConfigPath)
	Ω(err).NotTo(HaveOccurred())

	return brokerConfig
}

func BuildAndLaunchBroker(brokerConfigName string) *gexec.Session {
	brokerPath := buildExecutable("github.com/pivotal-cf/cf-redis-broker/cmd/broker")
	return LaunchProcessWithBrokerConfig(brokerPath, brokerConfigName)
}

func buildExecutable(sourcePath string) string {
	executable, err := gexec.Build(sourcePath)
	if err != nil {
		log.Fatalf("executable %s could not be built: %s", sourcePath, err)
		os.Exit(1)
	}
	return executable
}

func LaunchProcessWithBrokerConfig(executablePath string, brokerConfigName string) *gexec.Session {
	brokerConfigFile, filePathErr := helpers.AssetPath(brokerConfigName)
	Ω(filePathErr).ToNot(HaveOccurred())

	os.Setenv("BROKER_CONFIG_PATH", brokerConfigFile)
	processCmd := exec.Command(executablePath)
	processCmd.Stdout = GinkgoWriter
	processCmd.Stderr = GinkgoWriter
	return runCommand(processCmd)
}

func runCommand(cmd *exec.Cmd) *gexec.Session {
	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Ω(err).NotTo(HaveOccurred())
	return session
}
