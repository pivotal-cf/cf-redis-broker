package helpers

import (
	"os"
	"path"
	"path/filepath"

	. "github.com/onsi/gomega"
)

func ResetTestDirs() {
	removeAndRecreateDir("/tmp/redis-data-dir")
	removeAndRecreateDir("/tmp/redis-log-dir")
	removeAndRecreateDir("/tmp/redis-config-dir")
}

func removeAndRecreateDir(path string) {
	err := os.RemoveAll(path)
	Ω(err).ShouldNot(HaveOccurred())
	err = os.MkdirAll(path, 0755)
	Ω(err).ShouldNot(HaveOccurred())
}

func AssetPath(filename string) string {
	path, err := filepath.Abs(path.Join("assets", filename))
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
