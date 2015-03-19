package configmigratorintegration

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf/cf-redis-broker/redisconf"
)

var _ = Describe("ConfigMigrator Intgration", func() {
	Context("when there is data to migrate", func() {
		It("migrates the data", func() {
			redisDataDir := "/tmp/redis-data-dir"
			os.RemoveAll(redisDataDir)
			os.Mkdir(redisDataDir, 0755)

			redisInstanceDir := path.Join(redisDataDir, "instance1")
			os.Mkdir(redisInstanceDir, 0755)
			copyOverFromAssets("redis-server.port", redisInstanceDir)
			copyOverFromAssets("redis-server.password", redisInstanceDir)
			copyOverFromAssets("redis.conf", redisInstanceDir)

			executablePath := buildExecutable("github.com/pivotal-cf/cf-redis-broker/cmd/configmigrator")
			session := launchProcessWithBrokerConfig(executablePath, "broker.yml")
			session.Wait(10 * time.Second)

			Expect(session.ExitCode()).To(Equal(0))
			_, err := os.Stat(path.Join(redisInstanceDir, "redis-server.port"))
			Expect(os.IsNotExist(err)).To(BeTrue())
			_, err = os.Stat(path.Join(redisInstanceDir, "redis-server.password"))
			Expect(os.IsNotExist(err)).To(BeTrue())

			redisConfValues, _ := redisconf.Load(path.Join(redisInstanceDir, "redis.conf"))
			Expect(redisConfValues.Get("port")).To(Equal("1234"))
			Expect(redisConfValues.Get("requirepass")).To(Equal("secret-password"))
		})
	})
})

func copyOverFromAssets(fileName, dir string) {
	data, _ := ioutil.ReadFile(assetPath(fileName))
	ioutil.WriteFile(path.Join(dir, fileName), data, 0644)
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
	brokerConfigFile := assetPath(brokerConfigName)

	os.Setenv("BROKER_CONFIG_PATH", brokerConfigFile)
	processCmd := exec.Command(executablePath)
	processCmd.Stdout = GinkgoWriter
	processCmd.Stderr = GinkgoWriter
	return runCommand(processCmd)
}

func assetPath(filename string) string {
	path, _ := filepath.Abs(path.Join("assets", filename))
	return path
}

func runCommand(cmd *exec.Cmd) *gexec.Session {
	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Ω(err).NotTo(HaveOccurred())
	return session
}
