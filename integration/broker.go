package integration

import (
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

func BuildBroker() string {
	return helpers.BuildExecutable("github.com/pivotal-cf/cf-redis-broker/cmd/broker")
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
