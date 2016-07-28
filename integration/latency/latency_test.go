package latency_test

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf/cf-redis-broker/integration"
	"github.com/pivotal-cf/redis-backups/integration/helpers"
)

var session *gexec.Session
var cmd *exec.Cmd

var _ = Describe("Latency", func() {
	AfterEach(func() {
		session.Terminate()
		Eventually(session).Should(gexec.Exit())
	})

	JustBeforeEach(func() {
		var err error
		session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("when no redis config file is provided", func() {
		BeforeEach(func() {
			cmd = exec.Command(latencyExecutablePath)
		})

		It("Exits with status 2", func() {
			Eventually(session).Should(gexec.Exit(2))
		})

		It("logs that no config file was provided", func() {
			Eventually(session.Out).Should(gbytes.Say("No Redis config file provided"))
		})
	})

	Context("when no redis server is running on configured port", func() {
		var redisPort int

		BeforeEach(func() {
			redisPort = 3481
			redisTemplateData := &RedisTemplateData{
				RedisPort: redisPort,
			}

			confTemplate, err := filepath.Abs(filepath.Join("assets", "redis.conf.template"))
			Expect(err).NotTo(HaveOccurred())

			err = helpers.HandleTemplate(
				confTemplate,
				redisConfigFilePath,
				redisTemplateData,
			)
			Expect(err).ToNot(HaveOccurred())

			cmd = exec.Command(latencyExecutablePath, "-redisconf", redisConfigFilePath, "-config", "nothing")
		})

		It("Exits with status 2", func() {
			Eventually(session).Should(gexec.Exit(2))
		})

		It("logs that the connection to the host and port failed", func() {
			Eventually(session.Out).Should(gbytes.Say(fmt.Sprintf("dial tcp 127.0.0.1:%s:", strconv.Itoa(redisPort))))
			Eventually(session.Out).Should(gbytes.Say("connection refused"))
		})
	})

	Context("when valid configs are provided", func() {
		BeforeEach(func() {
			redisTemplateData := &RedisTemplateData{
				RedisPort: integration.RedisPort,
			}

			confTemplate, err := filepath.Abs(filepath.Join("assets", "redis.conf.template"))
			Expect(err).NotTo(HaveOccurred())

			err = helpers.HandleTemplate(
				confTemplate,
				redisConfigFilePath,
				redisTemplateData,
			)
			Expect(err).ToNot(HaveOccurred())

			latencyTemplateData := &LatencyTemplateData{
				LatencyFilePath: latencyFilePath,
				LatencyInterval: latencyInterval,
			}

			latencyConfTemplate, err := filepath.Abs(filepath.Join("assets", "latency.yml.template"))
			Expect(err).NotTo(HaveOccurred())

			err = helpers.HandleTemplate(
				latencyConfTemplate,
				latencyConfigFilePath,
				latencyTemplateData,
			)
			Expect(err).ToNot(HaveOccurred())

			cmd = exec.Command(
				latencyExecutablePath,
				"-redisconf", redisConfigFilePath,
				"-config", latencyConfigFilePath,
			)
		})

		It("logs that the monitor is starting", func() {
			Eventually(session.Out).Should(gbytes.Say("Starting Latency Monitor"))
		})

		It("logs when it is writing latency to file", func() {
			Eventually(session.Out, "2s").Should(gbytes.Say("Writing latency to file"))
		})

		It("writes output to the correct file", func() {
			Eventually(func() string {
				msg, _ := ioutil.ReadFile(latencyFilePath)
				return string(msg)
			}, "2s").Should(MatchRegexp(`\d.\d{2}`))
		})
	})

	Context("when no latency config file is provided", func() {
		BeforeEach(func() {
			redisTemplateData := &RedisTemplateData{
				RedisPort: integration.RedisPort,
			}

			confTemplate, err := filepath.Abs(filepath.Join("assets", "redis.conf.template"))
			Expect(err).NotTo(HaveOccurred())

			err = helpers.HandleTemplate(
				confTemplate,
				redisConfigFilePath,
				redisTemplateData,
			)
			Expect(err).ToNot(HaveOccurred())

			cmd = exec.Command(latencyExecutablePath, "-redisconf", redisConfigFilePath)
		})

		It("Exits with status 2", func() {
			Eventually(session).Should(gexec.Exit(2))
		})

		It("logs that no config file was provided", func() {
			Eventually(session.Out).Should(gbytes.Say("No Latency config file provided"))
		})
	})

	Context("when latency config file path is not a file", func() {
		BeforeEach(func() {
			redisTemplateData := &RedisTemplateData{
				RedisPort: integration.RedisPort,
			}

			confTemplate, err := filepath.Abs(filepath.Join("assets", "redis.conf.template"))
			Expect(err).NotTo(HaveOccurred())

			err = helpers.HandleTemplate(
				confTemplate,
				redisConfigFilePath,
				redisTemplateData,
			)
			Expect(err).ToNot(HaveOccurred())

			cmd = exec.Command(
				latencyExecutablePath,
				"-redisconf", redisConfigFilePath,
				"-config", "/not/a/file",
			)
		})

		It("Exits with status 2", func() {
			Eventually(session).Should(gexec.Exit(2))
		})

		It("logs that the config file does not exist", func() {
			Eventually(session.Out).Should(gbytes.Say("open /not/a/file: no such file or directory"))
		})
	})
})
