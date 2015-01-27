package redisconf_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf/cf-redis-broker/redisconf"
)

var _ = Describe("redisconf", func() {
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
				"save 300 10\n"

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
				Expect(conf.CommandAlias("CONFIG")).To(Equal("abc-def"))
			})
		})

		Context("when the command is aliased with quotes", func() {
			It("strips the quotes", func() {
				Expect(conf.CommandAlias("SAVE")).To(Equal("123-345"))
			})
		})

		Context("when the command is not alias", func() {
			It("returns the original command", func() {
				Expect(conf.CommandAlias("BGREWRITEAOF")).To(Equal("BGREWRITEAOF"))
			})
		})

		Context("when the command is disabled", func() {
			It("returns an empty string", func() {
				Expect(conf.CommandAlias("BGSAVE")).To(Equal(""))
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

	Describe("CopyWithSyslogAdditions", func() {
		It("writes the syslog configuration", func() {
			fromPath, err := filepath.Abs(path.Join("assets", "redis.conf"))
			Expect(err).ToNot(HaveOccurred())

			dir, err := ioutil.TempDir("", "redisconf-test")
			Expect(err).ToNot(HaveOccurred())
			toPath := filepath.Join(dir, "redis.conf")

			instanceID := "an-instance-id"

			err = redisconf.CopyWithSyslogAdditions(fromPath, toPath, instanceID)
			Ω(err).ToNot(HaveOccurred())

			resultingConf, err := redisconf.Load(toPath)
			Expect(err).ToNot(HaveOccurred())

			Ω(resultingConf.Get("syslog-enabled")).Should(Equal("yes"))
			Ω(resultingConf.Get("syslog-ident")).Should(Equal(fmt.Sprintf("redis-server-%s", instanceID)))
			Ω(resultingConf.Get("syslog-facility")).Should(Equal("local0"))
		})
	})
})
