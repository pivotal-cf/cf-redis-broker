package client_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/cf-redis-broker/redis/client"
	"github.com/pivotal-cf/cf-redis-broker/redisconf"

	"io/ioutil"
	"os"
	"os/exec"

	redisclient "github.com/garyburd/redigo/redis"
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

var host = "localhost"
var port uint = 6480
var password = ""

var _ = Describe("Client", func() {
	var redisArgs []string
	var redisRunner *RedisRunner
	var conf redisconf.Conf

	BeforeEach(func() {
		conf = redisconf.New()
		redisArgs = []string{"--port", "6480"}
	})

	Describe("connecting to a redis server", func() {
		Context("when the server is not running", func() {
			It("returns an error", func() {
				_, err := client.Connect(host, port, password, conf)
				Ω(err).Should(MatchError("dial tcp 127.0.0.1:6480: connection refused"))
			})
		})

		Context("when the server is running", func() {
			JustBeforeEach(func() {
				redisRunner = &RedisRunner{}
				redisRunner.Start(redisArgs)
			})

			AfterEach(func() {
				redisRunner.Stop()
			})

			It("connects with no error", func() {
				_, err := client.Connect(host, port, password, conf)
				Ω(err).ShouldNot(HaveOccurred())
			})

			Context("when the server has authentication enabled", func() {
				BeforeEach(func() {
					redisArgs = append(redisArgs, "--requirepass", "hello")
				})

				It("returns an error if the password is incorrect", func() {
					password := "goodbye"

					_, err := client.Connect(host, port, password, conf)
					Ω(err).Should(MatchError("ERR invalid password"))
				})

				It("works if the password is correct", func() {
					password := "hello"

					_, err := client.Connect(host, port, password, conf)
					Ω(err).ShouldNot(HaveOccurred())
				})
			})
		})
	})

	Describe("using the client", func() {
		BeforeEach(func() {
			redisRunner = &RedisRunner{}
			redisRunner.Start(redisArgs)
		})

		AfterEach(func() {
			redisRunner.Stop()
		})

		Describe("turning on appendonly", func() {
			It("turns on appendonly", func() {
				client, err := client.Connect(host, port, password, conf)
				Ω(err).ShouldNot(HaveOccurred())

				err = client.EnableAOF()
				Ω(err).ShouldNot(HaveOccurred())

				conn, err := redisclient.Dial("tcp", ":6480")
				Ω(err).ShouldNot(HaveOccurred())
				defer conn.Close()

				response, err := redisclient.Strings(conn.Do("CONFIG", "GET", "appendonly"))
				Ω(err).ShouldNot(HaveOccurred())

				Ω(response[1]).Should(Equal("yes"))
			})
		})

		Describe("querying info fields", func() {
			Context("when the field exits", func() {
				It("returns the value", func() {
					client, err := client.Connect(host, port, password, conf)
					Ω(err).ShouldNot(HaveOccurred())

					result, err := client.InfoField("aof_enabled")
					Ω(err).ShouldNot(HaveOccurred())
					Ω(result).To(Equal("0"))
				})
			})

			Context("when the field does not exist", func() {
				It("returns an error", func() {
					client, err := client.Connect(host, port, password, conf)
					Ω(err).ShouldNot(HaveOccurred())

					_, err = client.InfoField("made_up_field")
					Ω(err).Should(MatchError("Unknown field: made_up_field"))
				})
			})
		})
	})
})
