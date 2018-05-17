package redisconf_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf/cf-redis-broker/redisconf"
)

var _ = Describe("redisconf", func() {
	Describe("InitForDedicatedNode", func() {
		var conf redisconf.Conf

		BeforeEach(func() {
			path, err := filepath.Abs(path.Join("assets", "redis.conf"))
			Expect(err).ToNot(HaveOccurred())
			conf, err = redisconf.Load(path)
			Expect(err).ToNot(HaveOccurred())
			Expect(conf.Get("maxmemory")).To(BeEmpty())
			Expect(conf.Password()).To(BeEmpty())
		})

		It("sets the max memory parameter", func() {
			err := conf.InitForDedicatedNode()
			Expect(err).ToNot(HaveOccurred())

			maxmemory := conf.Get("maxmemory")
			Expect(maxmemory).ToNot(BeEmpty())

			_, err = strconv.Atoi(maxmemory)
			Expect(err).ToNot(HaveOccurred())
		})

		Context("called without password", func() {
			It("sets a random password", func() {
				err := conf.InitForDedicatedNode()
				Expect(err).ToNot(HaveOccurred())

				Expect(conf.Password()).ToNot(BeEmpty())
			})
		})

		Context("called with password", func() {
			It("sets the passed password", func() {
				err := conf.InitForDedicatedNode("my-password")
				Expect(err).ToNot(HaveOccurred())

				Expect(conf.Password()).To(Equal("my-password"))
			})
		})

		Context("called with multiple password", func() {
			It("returns an error", func() {
				err := conf.InitForDedicatedNode("my-password1", "my-password2")
				Expect(err).To(MatchError("Passed more than one password"))
			})
		})
	})

	Describe("Encode", func() {
		It("encodes the parameters correctly", func() {
			path, err := filepath.Abs(path.Join("assets", "redis.conf"))
			Expect(err).ToNot(HaveOccurred())
			input, err := redisconf.Load(path)
			Expect(err).ToNot(HaveOccurred())

			expectedOutput := "daemonize no\n" +
				"pidfile /var/run/redis.pid\n" +
				"port 6379\n" +
				"appendonly yes\n" +
				"client-output-buffer-limit normal 0 0 0\n" +
				"save 900 1\n" +
				"save 300 10\n" +
				"bind 0.0.0.0\n"

			Expect(string(input.Encode())).To(Equal(expectedOutput))
		})
	})

	Describe("CommandAlias", func() {
		conf := redisconf.New(
			redisconf.Param{Key: "rename-command", Value: "CONFIG abc-def"},
			redisconf.Param{Key: "rename-command", Value: "SAVE \"123-345\""},
			redisconf.Param{Key: "rename-command", Value: "BGSAVE \"\""},
		)

		Context("when the command is aliased", func() {
			It("returns the alias", func() {
				Expect(conf.CommandAliases()["CONFIG"]).To(Equal("abc-def"))
			})
		})

		Context("when the command is aliased with quotes", func() {
			It("strips the quotes", func() {
				Expect(conf.CommandAliases()["SAVE"]).To(Equal("123-345"))
			})
		})

		Context("when the command is disabled", func() {
			It("returns an empty string", func() {
				Expect(conf.CommandAliases()["BGSAVE"]).To(Equal(""))
			})
		})
	})

	Describe("Save", func() {
		conf := redisconf.New(
			redisconf.Param{Key: "client-output-buffer-limit", Value: "normal 0 0 0"},
			redisconf.Param{Key: "save", Value: "900 1"},
			redisconf.Param{Key: "save", Value: "300 10"},
		)

		Context("With an invalid path", func() {
			It("returns an error", func() {
				err := conf.Save("/this/is/an/invalid/path")
				Expect(err.Error()).To(Equal("open /this/is/an/invalid/path: no such file or directory"))
			})
		})

		Context("With a valid path", func() {
			It("saves the conf successfully", func() {
				dir, err := ioutil.TempDir("", "redisconf-test")
				path := filepath.Join(dir, "redis.conf")

				err = conf.Save(path)
				Expect(err).ToNot(HaveOccurred())

				loadedConf, err := redisconf.Load(path)
				Expect(err).ToNot(HaveOccurred())
				Expect(loadedConf).To(Equal(conf))

				os.RemoveAll(dir)
			})
		})
	})

	Describe("Load", func() {
		Context("When the file exists", func() {
			It("decodes all parameters", func() {
				path, err := filepath.Abs(path.Join("assets", "redis.conf"))
				Expect(err).ToNot(HaveOccurred())

				conf, err := redisconf.Load(path)
				Expect(err).ToNot(HaveOccurred())

				Expect(conf.Get("daemonize")).To(Equal("no"))
				Expect(conf.Get("appendonly")).To(Equal("yes"))
			})
		})

		Context("when the file is not valid", func() {
			It("returns an error", func() {
				invalidRedisConf, err := filepath.Abs(path.Join("assets", "invalid.conf"))
				Expect(err).ToNot(HaveOccurred())
				_, err = redisconf.Load(invalidRedisConf)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("When the file does not exist", func() {
			It("returns an error", func() {
				_, err := redisconf.Load("/this/is/an/invalid/path")
				Expect(err.Error()).To(Equal("open /this/is/an/invalid/path: no such file or directory"))
			})
		})
	})

	Describe("Set", func() {
		Context("When the key exists", func() {
			It("Sets the new value", func() {
				conf := redisconf.New(
					redisconf.Param{Key: "daemonize", Value: "yes"},
					redisconf.Param{Key: "save", Value: "900 1"},
				)

				conf.Set("daemonize", "no")
				Expect(conf.Get("daemonize")).To(Equal("no"))
			})
		})

		Context("When the key does not exist", func() {
			It("Inserts the new key/value pair", func() {
				conf := redisconf.New(
					redisconf.Param{Key: "daemonize", Value: "yes"},
					redisconf.Param{Key: "save", Value: "900 1"},
				)

				conf.Set("appendonly", "yes")
				Expect(conf.Get("appendonly")).To(Equal("yes"))
			})
		})
	})

	Describe("CopyWithInstanceAdditions", func() {
		var (
			copyErr       error
			resultingConf redisconf.Conf
			dir           string
			instanceID    = "an-instance-id"
			port          = "1234"
			password      = "an-password"
		)

		BeforeEach(func() {
			fromPath := absPath(path.Join("assets", "redis.conf"))
			dir = tempDir("", "redisconf-test")
			toPath := filepath.Join(dir, "redis.conf")

			copyErr = redisconf.CopyWithInstanceAdditions(fromPath, toPath, instanceID, port, password, dir)
			resultingConf = loadRedisConf(toPath)
		})

		It("does not return an error", func() {
			Expect(copyErr).NotTo(HaveOccurred())
		})

		It("writes the instance configuration", func() {
			Expect(resultingConf.Get("syslog-enabled")).To(Equal("yes"))
			Expect(resultingConf.Get("syslog-ident")).To(Equal(fmt.Sprintf("redis-server-%s", instanceID)))
			Expect(resultingConf.Get("syslog-facility")).To(Equal("local0"))
			Expect(resultingConf.Get("port")).To(Equal(port))
			Expect(resultingConf.Get("requirepass")).To(Equal(password))
			Expect(resultingConf.Get("pidfile")).To(Equal(filepath.Join(dir, instanceID+".pid")))
		})

		It("restricts the file permissions", func() {
			info, _ := os.Stat(filepath.Join(dir, "redis.conf"))
			fmt.Println(filepath.Join(dir, "redis.conf"))
			Expect(getPermissions(int(info.Mode()))).To(Equal(0640))
		})
	})
})

func getPermissions(fileMode int) int {
	return int(os.FileMode(fileMode).Perm())
}
