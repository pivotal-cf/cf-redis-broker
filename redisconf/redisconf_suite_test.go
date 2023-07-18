package redisconf_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/cf-redis-broker/redisconf"
)

func TestRedisconf(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Redisconf Suite")
}

func tempDir(dir, prefix string) string {
	temp, err := ioutil.TempDir(dir, prefix)
	Expect(err).ToNot(HaveOccurred())
	return temp
}

func absPath(path string) string {
	abs, err := filepath.Abs(path)
	Expect(err).ToNot(HaveOccurred())
	return abs
}

func loadRedisConf(path string) redisconf.Conf {
	conf, err := redisconf.Load(path)
	Expect(err).ToNot(HaveOccurred())
	return conf
}
