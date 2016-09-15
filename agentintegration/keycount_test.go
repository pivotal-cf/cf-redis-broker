package agentintegration_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"github.com/garyburd/redigo/redis"
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
			count, err := getKeyCount()
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
			count, err := getKeyCount()
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(2))
		})
	})
})

func getKeyCount() (int, error) {
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

	numKeysResponse := &struct {
		KeyCount int `json:"key_count"`
	}{}

	err = json.NewDecoder(response.Body).Decode(numKeysResponse)
	if err != nil {
		return 0, err
	}

	return numKeysResponse.KeyCount, nil
}
