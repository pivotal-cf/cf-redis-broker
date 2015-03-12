package api_test

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"

	"github.com/pivotal-cf/cf-redis-broker/api"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type fakeRedisResetter struct {
	deleteAllData func() error
}

func (client *fakeRedisResetter) ResetRedis() error {
	return client.deleteAllData()
}

var _ = Describe("redis agent HTTP API", func() {
	var server *httptest.Server
	var redisClient *fakeRedisResetter
	var deleteCount int
	var configPath string
	var response *http.Response

	BeforeEach(func() {
		var err error
		configPath, err = filepath.Abs("assets/redis.conf")
		Ω(err).ShouldNot(HaveOccurred())
		redisClient = &fakeRedisResetter{}
		deleteCount = 0
	})

	JustBeforeEach(func() {
		handler := api.New(redisClient, configPath)
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
				body, err := ioutil.ReadAll(response.Body)
				Ω(err).ShouldNot(HaveOccurred())

				response := map[string]interface{}{}
				err = json.Unmarshal(body, &response)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(response["port"]).Should(Equal(float64(1234))) // json.Unmarshal provides float64s by default
				Ω(response["password"]).Should(Equal("an-password"))
			})
		})

		Context("When it is unable to read the conf file", func() {
			BeforeEach(func() {
				configPath = "some/path/that/makes/no/sense"
			})

			It("returns an 500", func() {
				Ω(response.StatusCode).Should(Equal(500))
			})

			It("returns the correct error in the body", func() {
				body, err := ioutil.ReadAll(response.Body)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(string(body)).Should(ContainSubstring("no such file or directory"))
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
				Ω(deleteCount).To(Equal(1))
			})

			It("returns HTTP 200 OK", func() {
				Ω(response.StatusCode).Should(Equal(200))
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
				Ω(response.StatusCode).Should(Equal(500))
			})

			It("returns the correct error in the body", func() {
				body, err := ioutil.ReadAll(response.Body)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(string(body)).Should(Equal("redis burned down\n"))
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
				Ω(response.StatusCode).Should(Equal(http.StatusNotFound))
			})
		}
	})
})

func makeRequest(method string, url string) *http.Response {
	request, err := http.NewRequest(method, url, nil)
	Ω(err).ShouldNot(HaveOccurred())

	response, err := http.DefaultClient.Do(request)
	Ω(err).ShouldNot(HaveOccurred())

	return response
}
