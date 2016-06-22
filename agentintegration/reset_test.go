package agentintegration_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf/cf-redis-broker/integration/helpers"
	"github.com/pivotal-cf/cf-redis-broker/redisconf"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var redisSession *gexec.Session
var agentSession *gexec.Session

var _ = Describe("DELETE /", func() {

	var (
		redisConn         redis.Conn
		aofPath           string
		originalRedisConf redisconf.Conf
	)

	BeforeEach(func() {
		agentSession = startAgent()
		redisSession, aofPath = startRedisAndBlockUntilUp()

		var err error
		originalRedisConf, err = redisconf.Load(redisConfPath)
		Ω(err).ShouldNot(HaveOccurred())

		redisRestarted := make(chan bool)
		httpRequestReturned := make(chan bool)

		go checkRedisStopAndStart(redisRestarted)
		go doResetRequest(httpRequestReturned)

		select {
		case <-redisRestarted:
			<-httpRequestReturned
		case <-time.After(time.Second * 10):
			Fail("Test timed out after 10 seconds")
		}

		conf, err := redisconf.Load(redisConfPath)
		Ω(err).ShouldNot(HaveOccurred())

		redisConn = helpers.BuildRedisClientFromConf(conf)
	})

	AfterEach(func() {
		stopAgent(agentSession)
		stopRedisAndDeleteData(redisConn, aofPath)
	})

	It("no longer uses the original password", func() {
		password := originalRedisConf.Get("requirepass")
		port := originalRedisConf.Get("port")
		uri := fmt.Sprintf("127.0.0.1:%s", port)
		redisConn, err := redis.Dial("tcp", uri)
		Ω(err).ShouldNot(HaveOccurred())

		_, err = redisConn.Do("AUTH", password)
		Ω(err).Should(MatchError("ERR invalid password"))
	})

	It("resets the configuration", func() {
		config, err := redis.Strings(redisConn.Do("CONFIG", "GET", "maxmemory-policy"))

		Ω(err).ShouldNot(HaveOccurred())
		Ω(config[1]).Should(Equal("noeviction"))
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
	Expect(helpers.FileExists(aofPath)).To(BeTrue())

	return session, aofPath
}

func redisNotWritingAof(redisConn redis.Conn) func() bool {
	return func() bool {
		out, _ := redis.String(redisConn.Do("INFO", "persistence"))
		return strings.Contains(out, "aof_pending_rewrite:0") &&
			strings.Contains(out, "aof_rewrite_scheduled:0") &&
			strings.Contains(out, "aof_rewrite_in_progress:0")
	}
}

func doResetRequest(c chan<- bool) {
	defer GinkgoRecover()

	request, _ := http.NewRequest("DELETE", "http://127.0.0.1:9876", nil)
	request.SetBasicAuth("admin", "supersecretpassword")
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

	port, err := strconv.Atoi(conf.Get("port"))
	Ω(err).ShouldNot(HaveOccurred())

	Expect(helpers.ServiceAvailable(uint(port))).To(BeTrue())

	c <- true
}

func startRedis(confPath string) (*gexec.Session, redis.Conn) {
	redisSession, err := gexec.Start(exec.Command("redis-server", confPath), GinkgoWriter, GinkgoWriter)
	Ω(err).ShouldNot(HaveOccurred())

	conf, err := redisconf.Load(confPath)
	Ω(err).ShouldNot(HaveOccurred())

	port, err := strconv.Atoi(conf.Get("port"))
	Ω(err).ShouldNot(HaveOccurred())

	Expect(helpers.ServiceAvailable(uint(port))).To(BeTrue())
	return redisSession, helpers.BuildRedisClient(uint(port), "localhost", conf.Get("requirepass"))
}

func stopRedisAndDeleteData(redisConn redis.Conn, aofPath string) {
	redisSession.Kill().Wait()
	Eventually(redisSession).Should(gexec.Exit())

	os.Remove(aofPath)
	os.Remove(filepath.Join(aofPath, "..", "dump.rdb"))
}
