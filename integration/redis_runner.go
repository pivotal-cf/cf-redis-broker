package integration

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	redisclient "github.com/garyburd/redigo/redis"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/cf-redis-broker/integration/helpers"
)

type RedisRunner struct {
	process *os.Process
	Dir     string
	Port    uint
}

func NewRedisRunner(port uint) *RedisRunner {
	return &RedisRunner{
		Port: port,
	}
}

const RedisPort = 6480

func (runner *RedisRunner) Start(redisArgs []string) {
	command := exec.Command("redis-server", redisArgs...)

	if runner.Port == 0 {
		runner.Port = RedisPort
	}

	var err error
	runner.Dir, err = ioutil.TempDir("", "redis-client-test")
	Ω(err).ShouldNot(HaveOccurred())
	command.Dir = runner.Dir

	err = command.Start()
	Ω(err).ShouldNot(HaveOccurred())

	runner.process = command.Process

	Expect(helpers.ServiceAvailable(runner.Port)).To(BeTrue())
}

func (runner *RedisRunner) Stop() {
	err := runner.process.Kill()
	Ω(err).ShouldNot(HaveOccurred())

	Eventually(func() error {
		_, err = redisclient.Dial("tcp", fmt.Sprintf(":%d", runner.Port))
		return err
	}).Should(HaveOccurred())

	err = os.RemoveAll(runner.Dir)
	Ω(err).ShouldNot(HaveOccurred())
}
