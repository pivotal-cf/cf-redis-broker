package agentintegration_test

import (
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf/cf-redis-broker/redisconf"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var redisSession *gexec.Session
var agentSession *gexec.Session

var _ = Describe("DELETE /", func() {

	var redisConn redis.Conn
	var aofPath string

	BeforeEach(func() {
		agentSession = startAgentWithDefaultConfig()
		redisSession, aofPath = startRedisAndBlockUntilUp()

		redisRestarted := make(chan bool)
		httpRequestReturned := make(chan bool)

		go checkRedisStopAndStart(redisRestarted)
		go doResetRequest(httpRequestReturned)

		select {
		case <-redisRestarted:
			<-httpRequestReturned
		case <-httpRequestReturned:
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
		stopAgent(agentSession)
		stopRedisAndDeleteData(redisConn, aofPath)
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

func startRedisAndBlockUntilUp() (*gexec.Session, string) {
	session, connection := startRedis(redisConfPath)

	_, err := connection.Do("SET", "TEST-KEY", "TEST-VALUE")
	Ω(err).ShouldNot(HaveOccurred())

	_, err = connection.Do("CONFIG", "SET", "maxmemory-policy", "allkeys-lru")
	Ω(err).ShouldNot(HaveOccurred())

	cwd, err := os.Getwd()
	Ω(err).ShouldNot(HaveOccurred())
	aofPath := filepath.Join(cwd, "appendonly.aof")

	Eventually(redisNotWritingAof(connection)).Should(BeTrue())
	Eventually(fileExistsChecker(aofPath)).Should(BeTrue())

	return session, aofPath
}

func doResetRequest(c chan<- bool) {
	defer GinkgoRecover()

	request, _ := http.NewRequest("DELETE", "http://127.0.0.1:9876", nil)
	request.SetBasicAuth("admin", "secret")
	response, err := http.DefaultClient.Do(request)
	Ω(err).ShouldNot(HaveOccurred())
	Ω(response.StatusCode).To(Equal(http.StatusOK))

	c <- true
}

func checkRedisStopAndStart(c chan<- bool) {
	defer GinkgoRecover()

	Eventually(redisSession, "3s").Should(gexec.Exit())

	// Sleep here to emulate the time it takes monit to do it's thing
	time.Sleep(time.Millisecond * 200)

	var err error
	redisSession, err = gexec.Start(exec.Command("redis-server", redisConfPath), GinkgoWriter, GinkgoWriter)
	Ω(err).ShouldNot(HaveOccurred())

	conf, err := redisconf.Load(redisConfPath)
	Ω(err).ShouldNot(HaveOccurred())

	Eventually(func() error {
		_, err := buildRedisConn(conf)
		return err
	}).ShouldNot(HaveOccurred())

	c <- true
}

func stopRedisAndDeleteData(redisConn redis.Conn, aofPath string) {
	redisSession.Kill().Wait()
	Eventually(redisSession).Should(gexec.Exit())

	os.Remove(aofPath)
	os.Remove(filepath.Join(aofPath, "..", "dump.rdb"))
}
