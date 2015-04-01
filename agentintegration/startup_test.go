package agentintegration_test

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf/cf-redis-broker/agentconfig"
	"github.com/pivotal-cf/cf-redis-broker/redisconf"
)

var _ = Describe("Startup", func() {
	var session *gexec.Session
	var confPath string
	var config *agentconfig.Config

	BeforeEach(func() {
		defaultConfPath, err := filepath.Abs(path.Join("assets", "redis.conf.default"))
		Ω(err).ShouldNot(HaveOccurred())

		dir, err := ioutil.TempDir("", "redisconf-test")
		confPath = filepath.Join(dir, "redis.conf")

		config = &agentconfig.Config{
			DefaultConfPath: defaultConfPath,
			ConfPath:        confPath,
			Port:            "9876",
		}
	})

	Context("When redis.conf does not exist", func() {
		BeforeEach(func() {
			session = startAgentWithConfig(config)
			Eventually(listening("localhost:9876")).Should(BeTrue())
		})

		AfterEach(func() {
			session.Terminate().Wait()
			Eventually(session).Should(gexec.Exit())
		})

		It("Copies redis.conf from the default path and adds a password", func() {
			Eventually(fileExistsChecker(confPath)).Should(BeTrue())

			conf, err := redisconf.Load(confPath)
			Expect(err).ToNot(HaveOccurred())

			Expect(conf.Get("daemonize")).To(Equal("no"))
			Expect(conf.HasKey("requirepass")).To(BeTrue())
		})

		It("Creates a new password each time", func() {
			Eventually(fileExistsChecker(confPath)).Should(BeTrue())

			conf, err := redisconf.Load(confPath)
			Expect(err).ToNot(HaveOccurred())

			firstPassword := conf.Get("requirepass")

			session.Terminate().Wait()
			Eventually(session).Should(gexec.Exit())

			err = os.Remove(confPath)
			Expect(err).ToNot(HaveOccurred())

			session = startAgentWithConfig(config)
			Eventually(listening("localhost:9876")).Should(BeTrue())

			Eventually(fileExistsChecker(confPath)).Should(BeTrue())

			conf, err = redisconf.Load(confPath)
			Expect(err).ToNot(HaveOccurred())

			secondPassword := conf.Get("requirepass")

			Expect(firstPassword).NotTo(Equal(secondPassword))
		})
	})

	Context("When redis.conf already exists", func() {
		BeforeEach(func() {
			existingConf := redisconf.New(
				redisconf.Param{Key: "daemonize", Value: "yes"},
				redisconf.Param{Key: "requirepass", Value: "someotherpassword"},
				redisconf.Param{Key: "shouldbedeleted", Value: "yes"},
			)

			err := existingConf.Save(confPath)
			Expect(err).ToNot(HaveOccurred())

			session = startAgentWithConfig(config)
			Eventually(listening("localhost:9876")).Should(BeTrue())
		})

		AfterEach(func() {
			session.Terminate().Wait()
			Eventually(session).Should(gexec.Exit())
		})

		Describe("The copied redis.conf file", func() {
			var conf redisconf.Conf
			BeforeEach(func() {
				var err error
				conf, err = redisconf.Load(confPath)
				Expect(err).ToNot(HaveOccurred())
			})

			It("has it's original password", func() {
				Expect(conf.Get("requirepass")).To(Equal("someotherpassword"))
			})

			It("resets other parameters", func() {
				Expect(conf.Get("daemonize")).To(Equal("no"))
			})

			It("does not have additional parameters", func() {
				Expect(conf.HasKey("shouldbedeleted")).To(BeFalse())
			})
		})
	})
})
