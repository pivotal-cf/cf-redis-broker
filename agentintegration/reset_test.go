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
		redisConf redisconf.Conf
	)

	BeforeEach(func() {
		agentSession = startAgent()
		redisSession, aofPath = startRedisAndBlockUntilUp()

		var err error
		originalRedisConf, err = redisconf.Load(redisConfPath)
		Expect(err).NotTo(HaveOccurred())

		originalRedisConn := helpers.BuildRedisClientFromConf(originalRedisConf)
		_, err = originalRedisConn.Do("SET", "key", "val")
		Expect(err).NotTo(HaveOccurred())

		redisStopped := make(chan bool)
		go checkRedisStopped(redisStopped)
		sendResetRequest()

		select {
		case <-redisStopped:
			// Sleep here to emulate the time it takes monit to do it's thing
			time.Sleep(time.Millisecond * 200)

			redisSession, err = gexec.Start(exec.Command("redis-server", redisConfPath), GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())

		case <-time.After(time.Second * 10):
			Fail("Test timed out after 10 seconds")
		}

		redisConf = checkRedisConfigured(redisConfPath)
		redisConn = helpers.BuildRedisClientFromConf(redisConf)
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
		Expect(err).NotTo(HaveOccurred())

		_, err = redisConn.Do("AUTH", password)
		Expect(err).To(MatchError("ERR invalid password"))
	})

	It("resets the configuration", func() {
		config, err := redis.Strings(redisConn.Do("CONFIG", "GET", "maxmemory-policy"))

		Expect(err).NotTo(HaveOccurred())
		Expect(config[1]).To(Equal("noeviction"))
	})

	It("deletes all data from redis", func() {
		values, err := redis.Values(redisConn.Do("KEYS", "*"))
		Expect(err).NotTo(HaveOccurred())
		Expect(values).To(BeEmpty())
	})

	It("has an empty AOF file", func() {
		data, err := ioutil.ReadFile(aofPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(data)).To(Equal(""))
	})
})

func checkRedisConfigured(redisConfPath string) redisconf.Conf {
	conf, err := redisconf.Load(redisConfPath)
	Expect(err).NotTo(HaveOccurred())

	port, err := strconv.Atoi(conf.Get("port"))
	Expect(err).NotTo(HaveOccurred())

	Expect(helpers.ServiceAvailable(uint(port))).To(BeTrue())
	redisConf, err := redisconf.Load(redisConfPath)
	Expect(err).NotTo(HaveOccurred())

	return redisConf
}

func startRedisAndBlockUntilUp() (*gexec.Session, string) {
	session, connection := startRedis(redisConfPath)

	_, err := connection.Do("CONFIG", "SET", "maxmemory-policy", "allkeys-lru")
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

func sendResetRequest() {
	httpClient := &http.Client{
		Timeout:   300 * time.Second,
		Transport: http.DefaultTransport,
	}

	request, err := http.NewRequest("DELETE", "http://127.0.0.1:9876", nil)
	Expect(err).NotTo(HaveOccurred())

	request.SetBasicAuth("admin", "supersecretpassword")

	response, err := httpClient.Do(request)
	Expect(err).NotTo(HaveOccurred())

	defer response.Body.Close()
	bodyBytes, err := ioutil.ReadAll(response.Body)
	Expect(err).NotTo(HaveOccurred())
	fmt.Println(">>>>> HTTP Response from DELETE <<<<<<<<")
	fmt.Println(string(bodyBytes))
	fmt.Println("<<<<<<< END Response >>>>>>>")

	Expect(response.StatusCode).To(Equal(http.StatusOK))
}

func checkRedisStopped(c chan<- bool) {
	defer GinkgoRecover()
	Eventually(redisSession, "10s").Should(gexec.Exit())
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
