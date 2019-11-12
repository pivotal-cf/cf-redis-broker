package helpers

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/gomega"
)

const TestDataDir = "/tmp/redis-data-dir"
const TestLogDir = "/tmp/redis-log-dir"
const TestConfigDir = "/tmp/redis-config-dir"
const TestPidfileDir = "/tmp/pidfiles"

func ResetTestDirs() {
	removeAndRecreateDir(TestDataDir)
	removeAndRecreateDir(TestLogDir)
	removeAndRecreateDir(TestConfigDir)
	removeAndRecreateDir(TestPidfileDir)
}

func CreateTestDirs() (string, string, string) {
	var err error
	configDir, err := ioutil.TempDir("", "redis-config-")
	if err != nil {
		panic(err)
	}
	dataDir, err := ioutil.TempDir("", "redis-data-")
	if err != nil {
		panic(err)
	}
	logDir, err := ioutil.TempDir("", "redis-log-")
	if err != nil {
		panic(err)
	}
	return configDir, dataDir, logDir
}

func RemoveTestDirs(configDir, dataDir, logDir string) {
	err := os.RemoveAll(configDir)
	if err != nil {
		fmt.Printf("Error [%v] deleting config dir [%v] - delete manually!", err, configDir)
	}
	// Redis is weird
	os.Chmod(filepath.Join(dataDir, "instance1"), 0700)
	err = os.RemoveAll(dataDir)
	if err != nil {
		fmt.Printf("Error [%v] deleting data dir [%v] - delete manually!", err, dataDir)
	}
	err = os.RemoveAll(logDir)
	if err != nil {
		fmt.Printf("Error [%v] deleting log dir [%v] - delete manually!", err, logDir)
	}
}

func removeAndRecreateDir(path string) {
	err := os.RemoveAll(path)
	Ω(err).ShouldNot(HaveOccurred())
	err = os.MkdirAll(path, 0755)
	Ω(err).ShouldNot(HaveOccurred())
}

func AssetPath(filename string) string {
	path, err := filepath.Abs(filepath.Join("assets", filename))
	Ω(err).ShouldNot(HaveOccurred())
	return path
}

func FileExists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}
