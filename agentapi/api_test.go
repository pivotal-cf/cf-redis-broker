package agentapi_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf/cf-redis-broker/agentapi"
)

type fakeRedisResetter struct {
	deleteAllData func() error
}

func (client *fakeRedisResetter) ResetRedis() error {
	return client.deleteAllData()
}

var _ = Describe("redis agent HTTP API", func() {
	var (
		server      *httptest.Server
		redisClient *fakeRedisResetter
		deleteCount int
		configPath  string
		response    *http.Response
	)

	BeforeEach(func() {
		configPath = getAbsPath(filepath.FromSlash("assets/redis.conf"))
		redisClient = new(fakeRedisResetter)
		deleteCount = 0
	})

	JustBeforeEach(func() {
		handler := agentapi.New(redisClient, configPath)
		server = httptest.NewServer(handler)
	})

	AfterEach(func() {
		server.Close()
	})

	Describe("GET /", func() {
		JustBeforeEach(func() {
			response = makeRequest("GET", server.URL)
		})

		Context("When it can read the conf file successfully", func() {
			It("returns the correct credentials", func() {
				body := readAll(response.Body)
				response := unmarshalJSON(body)

				Expect(response["port"]).To(Equal(float64(1234)))
				Expect(response["password"]).To(Equal("an-password"))
			})
		})

		Context("When it is unable to read the conf file", func() {
			BeforeEach(func() {
				configPath = "some/path/that/makes/no/sense"
			})

			It("returns an 500", func() {
				Expect(response.StatusCode).To(Equal(http.StatusInternalServerError))
			})

			It("returns the correct error in the body", func() {
				body := string(readAll(response.Body))
				Expect(body).To(ContainSubstring("no such file or directory"))
			})
		})
	})

	Describe("DELETE /", func() {
		Context("When it can connect to Redis successfully", func() {
			JustBeforeEach(func() {
				redisClient.deleteAllData = func() error {
					deleteCount++
					return nil
				}

				response = makeRequest("DELETE", server.URL)
			})

			It("deletes all data from redis", func() {
				Expect(deleteCount).To(Equal(1))
			})

			It("returns HTTP 200 OK", func() {
				Expect(response.StatusCode).To(Equal(http.StatusOK))
			})
		})

		Context("when deleting all data from redis goes wrong", func() {
			JustBeforeEach(func() {
				redisClient.deleteAllData = func() error {
					return errors.New("redis burned down")
				}
				response = makeRequest("DELETE", server.URL)
			})

			It("returns 500", func() {
				Expect(response.StatusCode).To(Equal(http.StatusInternalServerError))
			})

			It("returns the correct error in the body", func() {
				body := string(readAll(response.Body))
				Expect(body).To(Equal("redis burned down\n"))
			})
		})
	})

	Describe("All other HTTP methods", func() {
		for _, method := range []string{"POST", "PUT"} {
			requestMethod := method
			var response *http.Response

			JustBeforeEach(func() {
				response = makeRequest(requestMethod, server.URL)
			})

			It(method+" returns an http error", func() {
				Expect(response.StatusCode).To(Equal(http.StatusNotFound))
			})
		}
	})
})

func makeRequest(method string, url string) *http.Response {
	request, err := http.NewRequest(method, url, nil)
	Expect(err).NotTo(HaveOccurred())

	response, err := http.DefaultClient.Do(request)
	Expect(err).NotTo(HaveOccurred())

	return response
}
