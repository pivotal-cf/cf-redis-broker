package client_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/cf-redis-broker/integration"
	"github.com/pivotal-cf/cf-redis-broker/redis/client"

	redisclient "github.com/garyburd/redigo/redis"
)

var host = "localhost"
var port = 6480
var password = ""
var pidFilePath string

var _ = Describe("Client", func() {
	var redisArgs []string
	var redisRunner *integration.RedisRunner

	BeforeEach(func() {
		pidFile, err := ioutil.TempFile("", "pid")
		Ω(err).ShouldNot(HaveOccurred())
		pidFilePath = pidFile.Name()
		redisArgs = []string{"--port", fmt.Sprintf("%d", port), "--pidfile", pidFilePath}
	})

	AfterEach(func() {
		os.Remove(pidFilePath)
	})

	Describe("connecting to a redis server", func() {
		Context("when the server is not running", func() {
			It("returns an error", func() {
				_, err := client.Connect(
					client.Host(host),
					client.Port(port),
				)
				Ω(err).Should(MatchError("dial tcp 127.0.0.1:6480: connection refused"))
			})
		})

		Context("when the server is running", func() {
			JustBeforeEach(func() {
				redisRunner = &integration.RedisRunner{}
				redisRunner.Start(redisArgs)
			})

			AfterEach(func() {
				redisRunner.Stop()
			})

			It("connects with no error", func() {
				_, err := client.Connect(client.Host(host), client.Port(port))
				Ω(err).ShouldNot(HaveOccurred())
			})

			Context("when the server has authentication enabled", func() {
				BeforeEach(func() {
					redisArgs = append(redisArgs, "--requirepass", "hello")
				})

				It("returns an error if the password is incorrect", func() {
					_, err := client.Connect(
						client.Host(host),
						client.Port(port),
						client.Password("wrong-password"),
					)
					Ω(err).Should(MatchError("ERR invalid password"))
				})

				It("works if the password is correct", func() {
					_, err := client.Connect(
						client.Host(host),
						client.Port(port),
						client.Password("hello"),
					)
					Ω(err).ShouldNot(HaveOccurred())
				})
			})
		})
	})

	Describe("using the client", func() {
		BeforeEach(func() {
			redisRunner = &integration.RedisRunner{}
			redisRunner.Start(redisArgs)
		})

		AfterEach(func() {
			redisRunner.Stop()
		})

		Describe("turning on appendonly", func() {
			It("turns on appendonly", func() {
				client, err := client.Connect(
					client.Host(host),
					client.Port(port),
				)
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

		Describe("creating a snapshot", func() {
			It("creates a snapshot", func() {
				client, err := client.Connect(
					client.Host(host),
					client.Port(port),
				)
				Ω(err).ShouldNot(HaveOccurred())

				beforeSnapshotLastSaveTime, err := client.LastRDBSaveTime()
				Ω(err).ShouldNot(HaveOccurred())

				err = client.CreateSnapshot(10)
				Ω(err).ShouldNot(HaveOccurred())

				afterSnapshotLastSaveTime, err := client.LastRDBSaveTime()
				Ω(err).ShouldNot(HaveOccurred())

				Ω(afterSnapshotLastSaveTime).Should(BeNumerically(">", beforeSnapshotLastSaveTime))
			})
		})

		Describe("querying info fields", func() {
			Context("when the field exits", func() {
				It("returns the value", func() {
					client, err := client.Connect(
						client.Host(host),
						client.Port(port),
					)
					Ω(err).ShouldNot(HaveOccurred())

					result, err := client.InfoField("aof_enabled")
					Ω(err).ShouldNot(HaveOccurred())
					Ω(result).To(Equal("0"))
				})
			})

			Context("when the field does not exist", func() {
				It("returns an error", func() {
					client, err := client.Connect(
						client.Host(host),
						client.Port(port),
					)
					Ω(err).ShouldNot(HaveOccurred())

					_, err = client.InfoField("made_up_field")
					Ω(err).Should(MatchError("Unknown field: made_up_field"))
				})
			})
		})
	})

	Describe(".GetConfig", func() {
		var redisClient client.Client

		BeforeEach(func() {
			redisRunner = &integration.RedisRunner{}
			redisRunner.Start(redisArgs)

			var err error
			redisClient, err = client.Connect(
				client.Host(host),
				client.Port(port),
			)
			Ω(err).ShouldNot(HaveOccurred())
		})

		AfterEach(func() {
			redisRunner.Stop()
		})

		Context("for a valid key", func() {
			It("returns the correct value", func() {
				actual, err := redisClient.GetConfig("port")
				Ω(err).ShouldNot(HaveOccurred())
				Ω(actual).Should(Equal(fmt.Sprintf("%d", port)))

				actual, err = redisClient.GetConfig("pidfile")
				Ω(err).ShouldNot(HaveOccurred())
				Ω(actual).Should(Equal(pidFilePath))
			})
		})

		Context("for an invalid key", func() {
			It("returns the an error", func() {
				_, err := redisClient.GetConfig("foobar")
				Ω(err).Should(MatchError("Key 'foobar' not found"))
			})
		})
	})

	Describe(".RDBPath", func() {
		var (
			redisClient  client.Client
			redisDataDir string
			redisRDBFile = "dump.rdb"
		)

		BeforeEach(func() {
			var err error
			redisDataDir, err = ioutil.TempDir("", "redisDataDir")
			Ω(err).ShouldNot(HaveOccurred())

			redisRunner = &integration.RedisRunner{}
			redisRunner.Start(append(redisArgs, "--dir", redisDataDir, "--dbfilename", redisRDBFile))

			redisClient, err = client.Connect(
				client.Host(host),
				client.Port(port),
			)
			Ω(err).ShouldNot(HaveOccurred())
		})

		AfterEach(func() {
			redisRunner.Stop()
		})

		It("returns the path to the RDB file", func() {
			path, _ := redisClient.RDBPath()

			var err error
			redisDataDir, err = filepath.EvalSymlinks(redisDataDir)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(path).Should(Equal(filepath.Join(redisDataDir, redisRDBFile)))
		})

		It("does not return an error", func() {
			_, err := redisClient.RDBPath()
			Ω(err).ShouldNot(HaveOccurred())
		})

		Context("when an error occurs", func() {
			BeforeEach(func() {
				redisRunner.Stop()
			})

			It("returns the error", func() {
				_, err := redisClient.RDBPath()
				Ω(err).Should(HaveOccurred())
			})
		})
	})
})
