package agentintegration_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf/cf-redis-broker/integration/helpers"
	"github.com/pivotal-cf/cf-redis-broker/redisconf"
)

var _ = Describe("Startup", func() {
	var agentSession *gexec.Session

	AfterEach(func() {
		stopAgent(agentSession)
	})

	Context("when there is no redis.conf", func() {
		BeforeEach(func() {
			os.Remove(redisConfPath)
			agentSession = startAgent()
		})

		loadRedisConfFileWhenItExists := func() redisconf.Conf {
			Eventually(func() bool {
				return helpers.FileExists(redisConfPath)
			}).Should(BeTrue())
			conf, err := redisconf.Load(redisConfPath)
			Expect(err).ToNot(HaveOccurred())
			return conf
		}

		It("copies a redis.conf from the default path and adds a password", func() {
			conf := loadRedisConfFileWhenItExists()

			Expect(conf.Get("daemonize")).To(Equal("no"))
			Expect(conf.HasKey("requirepass")).To(BeTrue())
		})

		It("creates a new password each time", func() {
			initialConf := loadRedisConfFileWhenItExists()

			helpers.KillProcess(agentSession)
			err := os.Remove(redisConfPath)
			Expect(err).ToNot(HaveOccurred())

			agentSession = startAgent()
			newConf := loadRedisConfFileWhenItExists()
			Expect(initialConf.Get("requirepass")).NotTo(Equal(newConf.Get("requirepass")))
		})
	})

	Context("when there is a redis.conf already", func() {
		BeforeEach(func() {
			err := redisconf.New(
				redisconf.Param{Key: "daemonize", Value: "yes"},
				redisconf.Param{Key: "requirepass", Value: "someotherpassword"},
				redisconf.Param{Key: "shouldbedeleted", Value: "yes"},
			).Save(redisConfPath)
			Expect(err).ToNot(HaveOccurred())
			agentSession = startAgent()
		})

		Describe("The copied redis.conf file", func() {
			It("has its original password", func() {
				newRedisConf, err := redisconf.Load(redisConfPath)
				Expect(err).ToNot(HaveOccurred())
				Expect(newRedisConf.Get("requirepass")).To(Equal("someotherpassword"))
			})

			It("resets other parameters", func() {
				newRedisConf, err := redisconf.Load(redisConfPath)
				Expect(err).ToNot(HaveOccurred())
				Expect(newRedisConf.Get("daemonize")).To(Equal("no"))
			})

			It("does not have additional parameters", func() {
				newRedisConf, err := redisconf.Load(redisConfPath)
				Expect(err).ToNot(HaveOccurred())
				Expect(newRedisConf.HasKey("shouldbedeleted")).To(BeFalse())
			})
		})
	})
})
