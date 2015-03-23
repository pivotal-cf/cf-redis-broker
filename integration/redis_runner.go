package integration

import (
	"io/ioutil"
	"os"
	"os/exec"

	redisclient "github.com/garyburd/redigo/redis"
	. "github.com/onsi/gomega"
)

type RedisRunner struct {
	process *os.Process
	dir     string
}

func (runner *RedisRunner) Start(redisArgs []string) {
	command := exec.Command("redis-server", redisArgs...)

	var err error
	runner.dir, err = ioutil.TempDir("", "redis-client-test")
	Ω(err).ShouldNot(HaveOccurred())
	command.Dir = runner.dir

	err = command.Start()
	Ω(err).ShouldNot(HaveOccurred())

	runner.process = command.Process

	Eventually(func() error {
		_, err := redisclient.Dial("tcp", ":6480")
		return err
	}).ShouldNot(HaveOccurred())
}

func (runner *RedisRunner) Stop() {
	err := runner.process.Kill()
	Ω(err).ShouldNot(HaveOccurred())

	Eventually(func() error {
		_, err := redisclient.Dial("tcp", ":6480")
		return err
	}).Should(HaveOccurred())

	err = os.RemoveAll(runner.dir)
	Ω(err).ShouldNot(HaveOccurred())
}
