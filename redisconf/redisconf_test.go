package redisconf_test

import (
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
			input := redisconf.New(
				redisconf.Param{Key: "client-output-buffer-limit", Value: "normal 0 0 0"},
				redisconf.Param{Key: "save", Value: "900 1"},
				redisconf.Param{Key: "save", Value: "300 10"},
			)

			expectedOutput := "client-output-buffer-limit normal 0 0 0\n" +
				"save 900 1\n" +
				"save 300 10\n"

			Expect(string(input.Encode())).To(Equal(expectedOutput))
		})
	})

	Describe("Decode", func() {
		It("decodes the file correctly", func() {
			input := "client-output-buffer-limit normal 0 0 0\n" +
				"# This is a comment\n" +
				"save 900 1\n" +
				"\n" +
				"\t\t\t   \n" +
				"save 300 10\n"

			expectedOutput := redisconf.New(
				redisconf.Param{Key: "client-output-buffer-limit", Value: "normal 0 0 0"},
				redisconf.Param{Key: "save", Value: "900 1"},
				redisconf.Param{Key: "save", Value: "300 10"},
			)

			output, err := redisconf.Decode([]byte(input))
			Expect(err).ToNot(HaveOccurred())
			Expect(output).To(Equal(expectedOutput))
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

	Describe("Decoding then encoding", func() {
		It("does not lose data, other than comments and empty lines", func() {
			input := "client-output-buffer-limit normal 0 0 0\n" +
				"# This is a comment\n" +
				"save 900 1\n" +
				"\n" +
				"\t\t\t   \n" +
				"save 300 10\n" +
				"# another comment"

			expectedOutput := "client-output-buffer-limit normal 0 0 0\n" +
				"save 900 1\n" +
				"save 300 10\n"

			conf, err := redisconf.Decode([]byte(input))
			Expect(err).ToNot(HaveOccurred())

			output := string(conf.Encode())
			Expect(output).To(Equal(expectedOutput))
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
})
