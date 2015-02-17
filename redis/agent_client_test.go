package redis_test

import (
	"net"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf/cf-redis-broker/brokerconfig"
	"github.com/pivotal-cf/cf-redis-broker/redis"
)

var _ = Describe("RemoteAgentClient", func() {
	var server *httptest.Server
	var agentCalled int
	var remoteAgentClient redis.RemoteAgentClient
	var status int

	const (
		hostAndPort = "127.0.0.1:8080"
		rootURL     = "http://127.0.0.1:8080"
	)

	BeforeEach(func() {
		remoteAgentClient = redis.RemoteAgentClient{
			HttpAuth: brokerconfig.AuthConfiguration{
				Username: "username",
				Password: "password",
			},
		}
		agentCalled = 0

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer GinkgoRecover()

			username, password, _ := r.BasicAuth()
			Expect(username).To(Equal(remoteAgentClient.HttpAuth.Username))
			Expect(password).To(Equal(remoteAgentClient.HttpAuth.Password))

			Ω([]string{"DELETE", "GET"}).Should(ContainElement(r.Method))
			Ω(r.URL.Path).Should(Equal("/"))
			agentCalled++
			w.WriteHeader(status)
			if r.Method == "GET" {
				w.Write([]byte("{\"port\": 12345, \"password\": \"super-secret\"}"))
			}
		})

		listener, err := net.Listen("tcp", hostAndPort)
		Ω(err).ShouldNot(HaveOccurred())

		server = httptest.NewUnstartedServer(handler)
		server.Listener = listener
		server.Start()
		Eventually(isListeningChecker(hostAndPort)).Should(BeTrue())
	})

	AfterEach(func() {
		server.Close()
		Eventually(isListeningChecker(hostAndPort)).Should(BeFalse())
	})

	Describe("#Reset", func() {
		Context("when the DELETE request is successful", func() {
			BeforeEach(func() {
				status = http.StatusOK
			})

			It("makes a DELETE request to the rootURL", func() {
				remoteAgentClient.Reset(rootURL)
				Ω(agentCalled).Should(Equal(1))
			})
		})

		Context("When the DELETE request fails", func() {
			BeforeEach(func() {
				status = http.StatusInternalServerError
			})

			It("returns the error", func() {
				err := remoteAgentClient.Reset(rootURL)
				Ω(err).To(MatchError("Agent error: 500"))
			})
		})
	})

	Describe("#Credentials", func() {
		BeforeEach(func() {
			status = http.StatusOK
		})

		It("makes a GET request to the rootURL", func() {
			remoteAgentClient.Credentials(rootURL)
			Ω(agentCalled).Should(Equal(1))
		})

		Context("When successful", func() {
			It("returns the correct credentials", func() {
				credentials, err := remoteAgentClient.Credentials(rootURL)
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
				_, err := remoteAgentClient.Credentials(rootURL)
				Ω(err).Should(HaveOccurred())
				Ω(err.Error()).Should(Equal(`Agent error: 500, {"port": 12345, "password": "super-secret"}`))
			})
		})
	})
})
