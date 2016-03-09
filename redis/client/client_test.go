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

				// on OS X, "getsockopt:" is also present in the error message from the system
				Ω(err).Should(MatchError(MatchRegexp("dial tcp 127.0.0.1:6480: (getsockopt: )?connection refused")))
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

	Describe(".Disconnect", func() {
		var (
			disconnectErr error
			redisClient   client.Client
		)

		BeforeEach(func() {
			redisRunner = &integration.RedisRunner{}
			redisRunner.Start(redisArgs)

			var err error
			redisClient, err = client.Connect(
				client.Host(host),
				client.Port(port),
			)
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			redisRunner.Stop()
		})

		JustBeforeEach(func() {
			disconnectErr = redisClient.Disconnect()
		})

		It("does not return an error", func() {
			Expect(disconnectErr).ToNot(HaveOccurred())
		})

		It("closes the redis connection", func() {
			_, err := redisClient.RDBPath()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("closed network connection"))
		})

		Context("when the client is not connected", func() {
			BeforeEach(func() {
				Expect(redisClient.Disconnect()).ToNot(HaveOccurred())
			})

			It("returns an error", func() {
				Expect(disconnectErr).To(HaveOccurred())
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

		Describe(".Info", func() {
			var redis client.Client

			BeforeEach(func() {
				var err error
				redis, err = client.Connect(
					client.Host(host),
					client.Port(port),
				)
				Expect(err).ToNot(HaveOccurred())
			})

			It("does not return an error", func() {
				_, err := redis.Info()
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns a map with multiple entries", func() {
				info, _ := redis.Info()
				Expect(len(info)).To(BeNumerically(">", 1))
			})

			It("returns a map that contains expected entries needed to emit as metrics", func() {
				info, _ := redis.Info()

				expectedInfoKeys := []string{
					"arch_bits",
					"uptime_in_seconds",
					"uptime_in_days",
					"hz",
					"lru_clock",
					"connected_clients",
					"client_longest_output_list",
					"client_biggest_input_buf",
					"blocked_clients",
					"used_memory",
					"used_memory_rss",
					"used_memory_peak",
					"used_memory_lua",
					"mem_fragmentation_ratio",
					"loading",
					"rdb_changes_since_last_save",
					"rdb_bgsave_in_progress",
					"rdb_last_save_time",
					"rdb_last_bgsave_time_sec",
					"rdb_current_bgsave_time_sec",
					"aof_rewrite_in_progress",
					"aof_rewrite_scheduled",
					"aof_last_rewrite_time_sec",
					"aof_current_rewrite_time_sec",
					"total_connections_received",
					"total_commands_processed",
					"instantaneous_ops_per_sec",
					"total_net_input_bytes",
					"total_net_output_bytes",
					"instantaneous_input_kbps",
					"instantaneous_output_kbps",
					"rejected_connections",
					"sync_full",
					"sync_partial_ok",
					"sync_partial_err",
					"expired_keys",
					"evicted_keys",
					"keyspace_hits",
					"keyspace_misses",
					"pubsub_channels",
					"pubsub_patterns",
					"latest_fork_usec",
					"migrate_cached_sockets",
					"connected_slaves",
					"master_repl_offset",
					"repl_backlog_active",
					"repl_backlog_size",
					"repl_backlog_first_byte_offset",
					"repl_backlog_histlen",
					"used_cpu_sys",
					"used_cpu_user",
					"used_cpu_sys_children",
					"used_cpu_user_children",
					"cluster_enabled",
				}

				actualInfoKeys := []string{}

				for key, _ := range info {
					actualInfoKeys = append(actualInfoKeys, key)
				}

				for _, expectedInfoKey := range expectedInfoKeys {
					Expect(actualInfoKeys).To(ContainElement(expectedInfoKey))
				}
			})

			It("should return db info so that we can parse this as metrics further down the line", func() {
				address := fmt.Sprintf("%v:%v", host, port)
				connection, err := redisclient.Dial("tcp", address)
				Expect(err).ToNot(HaveOccurred())
				connection.Do("SET", "foo", "bar")

				info, _ := redis.Info()

				Expect(info["db0"]).To(Equal("keys=1,expires=0,avg_ttl=0"))
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

	Describe(".Address", func() {
		var redisClient client.Client

		BeforeEach(func() {
			redisRunner = &integration.RedisRunner{}
			redisRunner.Start(redisArgs)
			var err error
			redisClient, err = client.Connect(client.Host(host), client.Port(port))
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			redisRunner.Stop()
		})

		It("returns an address that consists of the client's host and port", func() {
			Expect(redisClient.Address()).To(Equal(fmt.Sprintf("%s:%d", host, port)))
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
