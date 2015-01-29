package agentintegration_test

import (
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf/cf-redis-broker/agentconfig"
	"github.com/pivotal-cf/cf-redis-broker/redisconf"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DELETE /", func() {

	var session *gexec.Session
	var redisConfPath string
	var originalConf redisconf.Conf

	BeforeEach(func() {
		os.Remove("/tmp/fake_monit_start_stack")
		os.Remove("/tmp/fake_monit_start_stop")

		defaultConfPath, err := filepath.Abs(path.Join("assets", "redis.conf.default"))
		Ω(err).ShouldNot(HaveOccurred())

		dir, err := ioutil.TempDir("", "redisconf-test")
		redisConfPath = filepath.Join(dir, "redis.conf")

		originalConf = redisconf.New(
			redisconf.Param{Key: "requirepass", Value: "thepassword"},
			redisconf.Param{Key: "port", Value: "6379"},
		)

		config := &agentconfig.Config{
			DefaultConfPath:     defaultConfPath,
			ConfPath:            redisConfPath,
			MonitExecutablePath: "assets/fake_monit",
		}

		err = originalConf.Save(redisConfPath)
		Expect(err).ToNot(HaveOccurred())

		session = startAgentWithConfig(config)
		Eventually(isListeningChecker("localhost:9876")).Should(BeTrue())
	})

	AfterEach(func() {
		session.Terminate().Wait()
		Eventually(session).Should(gexec.Exit())

		redisConfPath, err := filepath.Abs(path.Join("assets", "redis.conf.tmp"))
		Ω(err).ShouldNot(HaveOccurred())
		os.Remove(redisConfPath)
	})

	Context("when redis is up after being reset", func() {

		var redisSession *gexec.Session
		var redisConn redis.Conn
		var aofPath string

		BeforeEach(func() {
			redisSession, redisConn = startRedis(redisConfPath)

			_, err := redisConn.Do("SET", "TEST-KEY", "TEST-VALUE")
			Ω(err).ShouldNot(HaveOccurred())

			_, err = redisConn.Do("CONFIG", "SET", "maxmemory-policy", "allkeys-lru")
			Ω(err).ShouldNot(HaveOccurred())

			cwd, err := os.Getwd()
			Ω(err).ShouldNot(HaveOccurred())
			aofPath = filepath.Join(cwd, "appendonly.aof")

			Eventually(redisNotWritingAof(redisConn)).Should(BeTrue())
			Eventually(fileExistsChecker(aofPath)).Should(BeTrue())

			redisRestarted := make(chan bool)
			httpRequestReturned := make(chan bool)

			go func(c chan<- bool) {
				defer GinkgoRecover()
				request, _ := http.NewRequest("DELETE", "http://127.0.0.1:9876", nil)
				response, err := http.DefaultClient.Do(request)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(response.StatusCode).To(Equal(http.StatusOK))
				c <- true
			}(httpRequestReturned)

			go func(c chan<- bool) {
				defer GinkgoRecover()
				Eventually(redisSession, "3s").Should(gexec.Exit())
				monitStartCmdListPath := "/tmp/fake_monit_start_stack"
				Eventually(func() string {
					contents, _ := ioutil.ReadFile(monitStartCmdListPath)
					return string(contents)
				}, "5s").Should(Equal("redis\n"))
				var err error
				redisSession, err = gexec.Start(exec.Command("redis-server", redisConfPath), GinkgoWriter, GinkgoWriter)
				Ω(err).ShouldNot(HaveOccurred())
				c <- true
			}(redisRestarted)

			select {
			case <-redisRestarted:
				<-httpRequestReturned
			case <-httpRequestReturned:
				<-redisRestarted
				Fail("DELETE request returned before redis had been restarted")
			case <-time.After(time.Second * 10):
				Fail("Test timed out after 10 seconds")
			}

			conf, err := redisconf.Load(redisConfPath)
			Ω(err).ShouldNot(HaveOccurred())
			redisConn, err = buildRedisConn(conf)
			Ω(err).ShouldNot(HaveOccurred())
		})

		AfterEach(func() {
			Eventually(redisNotWritingAof(redisConn)).Should(BeTrue())
			redisSession.Kill().Wait()
			Eventually(redisSession).Should(gexec.Exit())

			err := os.Remove(aofPath)
			Ω(err).ShouldNot(HaveOccurred())

			os.Remove(filepath.Join(aofPath, "..", "dump.rdb"))

			err = os.Remove("/tmp/fake_monit_start_stack")
			Ω(err).ShouldNot(HaveOccurred())
			err = os.Remove("/tmp/fake_monit_stop_stack")
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("no longer uses the original password", func() {
			_, err := buildRedisConn(originalConf)
			Ω(err).Should(MatchError("ERR invalid password"))
		})

		It("resets the configuration", func() {
			config, err := redis.Strings(redisConn.Do("CONFIG", "GET", "maxmemory-policy"))

			Ω(err).ShouldNot(HaveOccurred())
			Ω(config[1]).Should(Equal("volatile-lru"))
		})

		It("deletes all data from redis", func() {
			values, err := redis.Values(redisConn.Do("KEYS", "*"))
			Ω(err).ShouldNot(HaveOccurred())
			Ω(values).Should(BeEmpty())
		})

		It("has an empty AOF file", func() {
			data, err := ioutil.ReadFile(aofPath)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(string(data)).Should(Equal(""))
		})
	})

	Context("when there is some failure at the redis level", func() {
		It("responds with HTTP 503", func() {
			request, _ := http.NewRequest("DELETE", "http://127.0.0.1:9876", nil)

			response, err := http.DefaultClient.Do(request)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(response.StatusCode).To(Equal(503))
		})
	})
})
