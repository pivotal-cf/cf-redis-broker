package redis_test

import (
	"net"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf/cf-redis-broker/redis"
)

var _ = Describe("RemoteAgentClient", func() {
	var server *httptest.Server
	var agentCalled int
	var remoteAgentClient redis.RemoteAgentClient
	var status int

	BeforeEach(func() {
		remoteAgentClient = redis.RemoteAgentClient{}
		agentCalled = 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			Ω([]string{"DELETE", "GET"}).Should(ContainElement(r.Method))
			Ω(r.URL.Path).Should(Equal("/"))
			agentCalled++
			w.WriteHeader(status)
			if r.Method == "GET" {
				w.Write([]byte("{\"port\": 12345, \"password\": \"super-secret\"}"))
			}
		})

		server = httptest.NewUnstartedServer(handler)
		listener, err := net.Listen("tcp", "127.0.0.1:9876")
		Ω(err).ShouldNot(HaveOccurred())
		server.Listener = listener
		server.Start()
		Eventually(isListeningChecker("127.0.0.1:9876")).Should(BeTrue())
	})

	AfterEach(func() {
		server.Close()
		Eventually(isListeningChecker("127.0.0.1:9876")).Should(BeFalse())
	})

	Describe("#Reset", func() {
		Context("when the DELETE request is successful", func() {
			BeforeEach(func() {
				status = http.StatusOK
			})

			It("makes a DELETE request to http://<host ip>:9876/", func() {
				remoteAgentClient.Reset("127.0.0.1")
				Ω(agentCalled).Should(Equal(1))
			})
		})

		Context("When the DELETE request fails", func() {
			BeforeEach(func() {
				status = http.StatusInternalServerError
			})

			It("returns the error", func() {
				err := remoteAgentClient.Reset("127.0.0.1")
				Ω(err).To(MatchError("Expected status code 200, received 500"))
			})
		})
	})

	Describe("#Credentials", func() {
		BeforeEach(func() {
			status = http.StatusOK
		})

		It("makes a GET request to http://<host ip>:9876/", func() {
			remoteAgentClient.Credentials("127.0.0.1")
			Ω(agentCalled).Should(Equal(1))
		})

		Context("When successful", func() {
			It("returns the correct credentials", func() {
				credentials, err := remoteAgentClient.Credentials("127.0.0.1")
				Ω(err).ShouldNot(HaveOccurred())

				Ω(credentials).Should(Equal(redis.Credentials{
					Port:     12345,
					Password: "super-secret",
				}))
			})
		})

		Context("When unsuccessful", func() {
			It("returns an error", func() {
				status = http.StatusInternalServerError
				_, err := remoteAgentClient.Credentials("127.0.0.1")
				Ω(err).Should(HaveOccurred())
				Ω(err.Error()).Should(Equal("Received non-200 status code from agent"))
			})
		})
	})
})
