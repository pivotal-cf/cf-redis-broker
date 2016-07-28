package latency_test

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/pivotal-cf/cf-redis-broker/redis/client/fakes"
	. "github.com/pivotal-cf/cf-redis-broker/redis/latency"
	"github.com/pivotal-golang/lager"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Latency", func() {
	var (
		redisClient     *fakes.FakeClient
		latencyDir      string
		latencyFilePath string
		interval        time.Duration
		logger          lager.Logger
		log             *gbytes.Buffer

		latency *Latency
	)

	BeforeEach(func() {
		redisClient = new(fakes.FakeClient)

		var err error
		latencyDir, err = ioutil.TempDir("", "redis-latency-")
		Expect(err).ToNot(HaveOccurred())
		latencyFilePath = filepath.Join(latencyDir, "latency")

		interval, err = time.ParseDuration("1s")
		Expect(err).ToNot(HaveOccurred())

		logger = lager.NewLogger("latency-unit-test")
		log = gbytes.NewBuffer()
		logger.RegisterSink(lager.NewWriterSink(log, lager.INFO))

		latency = NewLatency(
			redisClient,
			latencyFilePath,
			interval,
			logger,
		)
	})

	JustBeforeEach(func() {
		latency.Start()
	})

	AfterEach(func() {
		latency.Stop()
		os.RemoveAll(latencyDir)
	})

	It("sends a ping to redis", func() {
		Eventually(redisClient.PingCallCount).Should(BeNumerically(">", 0))
	})

	It("writes avg latency to a file", func() {
		Eventually(func() string {
			contents, _ := ioutil.ReadFile(latencyFilePath)
			return string(contents)
		}, "2s").Should(MatchRegexp(`\d.\d{2}$`))
	})

	It("it writes to the file every iteration", func() {
		Eventually(func() error {
			_, err := os.Stat(latencyFilePath)
			return err
		}, "2s").ShouldNot(HaveOccurred())
		Eventually(func() time.Time {
			info, _ := os.Stat(latencyFilePath)
			return info.ModTime()
		}, "5s").ShouldNot(BeTemporally("~", time.Now(), interval))
	})

	It("logs when it starts monitoring", func() {
		Eventually(log, "2s").Should(gbytes.Say("Start latency monitering"))
	})

	It("logs when it is writing latency to file", func() {
		Eventually(log, "2s").Should(gbytes.Say("Writing latency to file"))
	})
})

var _ = Describe("Config", func() {
	Describe("LoadConfig", func() {
		Context("when the file does not exist", func() {
			It("returns an error", func() {
				_, err := LoadConfig("/this/is/not/a/file")
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when a valid config file is provided", func() {
			var config *Config

			BeforeEach(func() {
				path, err := filepath.Abs(path.Join("assets", "latency.yml"))
				Expect(err).NotTo(HaveOccurred())

				config, err = LoadConfig(path)
				Expect(err).ToNot(HaveOccurred())
			})

			It("has the correct interval", func() {
				Expect(config.Interval).To(Equal("5s"))
			})

			It("has the correct latency file path", func() {
				Expect(config.LatencyFilePath).To(Equal("/tmp/latency-file"))
			})
		})
	})
})
