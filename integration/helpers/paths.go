package helpers

import (
	"os"
	"path"
	"path/filepath"

	. "github.com/onsi/gomega"
)

func SafelyResetAllDirectories() {
	RemoveAndRecreateDir("/tmp/redis-data-dir")
	RemoveAndRecreateDir("/tmp/redis-log-dir")
	RemoveAndRecreateDir("/tmp/redis-config-dir")
}

func RemoveAndRecreateDir(path string) {
	err := os.RemoveAll(path)
	Ω(err).ShouldNot(HaveOccurred())
	err = os.MkdirAll(path, 0755)
	Ω(err).ShouldNot(HaveOccurred())
}

func AssetPath(filename string) (string, error) {
	return filepath.Abs(path.Join("assets", filename))
}
