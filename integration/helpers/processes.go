package helpers

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf/cf-redis-broker/brokerconfig"
)

func BuildExecutable(sourcePath string) string {
	executable, err := gexec.Build(sourcePath)
	if err != nil {
		log.Fatalf("executable %s could not be built: %s", sourcePath, err)
	}
	return executable
}

func KillProcess(session *gexec.Session) {
	session.Terminate().Wait()
	Eventually(session).Should(gexec.Exit())
}

func KillRedisProcess(instanceID string, brokerConfig brokerconfig.Config) {
	pidFilePath, err := filepath.Abs(path.Join(brokerConfig.RedisConfiguration.PidfileDirectory, instanceID+".pid"))
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
