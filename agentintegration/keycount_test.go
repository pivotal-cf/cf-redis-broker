package agentintegration_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pivotal-cf/cf-redis-broker/integration/helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"github.com/gomodule/redigo/redis"
	"github.com/pivotal-cf/cf-redis-broker/agentapi"
	"github.com/pivotal-cf/cf-redis-broker/redisconf"
)

var _ = Describe("keycount request", func() {
	var (
		agentSession, redisSession *gexec.Session
		aofPath                    string
		conn                       redis.Conn
	)

	BeforeEach(func() {
		agentSession = startAgent()
		redisSession, aofPath = startRedisAndBlockUntilUp()

		conf, err := redisconf.Load(redisConfPath)
		Expect(err).ToNot(HaveOccurred())

		conn, err = redis.Dial("tcp", fmt.Sprintf("127.0.0.1:%s", conf.Get("port")))
		Expect(err).ToNot(HaveOccurred())

		if password := conf.Get("requirepass"); password != "" {
			_, err = conn.Do("AUTH", password)
			Expect(err).ToNot(HaveOccurred())
		}

		_, err = conn.Do("FLUSHALL")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		agentSession.Kill()
		redisSession.Kill()
		os.Remove(aofPath)
	})

	Context("when the redis database is empty", func() {
		It("reports zero keys", func() {
			count, err := getKeycount()
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(0))
		})
	})

	Context("when the redis database contains two keys", func() {
		BeforeEach(func() {
			_, err := conn.Do("SET", "FOO", "BAR")
			Expect(err).ToNot(HaveOccurred())

			_, err = conn.Do("SET", "BAZ", "BAR")
			Expect(err).ToNot(HaveOccurred())
		})

		It("reports two keys", func() {
			count, err := getKeycount()
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(2))
		})
	})
})

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

func getKeycount() (int, error) {
	httpClient := &http.Client{
		Timeout:   5 * time.Second,
		Transport: http.DefaultTransport,
	}

	request, err := http.NewRequest("GET", "http://127.0.0.1:9876/keycount", nil)
	if err != nil {
		return 0, err
	}

	request.SetBasicAuth("admin", "supersecretpassword")

	response, err := httpClient.Do(request)
	if err != nil {
		return 0, err
	}

	if want, got := response.StatusCode, http.StatusOK; want != got {
		return 0, fmt.Errorf("unexpected HTTP response code: want %d, got %d", want, got)
	}

	keycountResp := new(agentapi.KeycountResponse)
	err = json.NewDecoder(response.Body).Decode(keycountResp)
	if err != nil {
		return 0, err
	}

	return keycountResp.Keycount, nil
}
