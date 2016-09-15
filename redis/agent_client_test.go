package redis_test

import (
	"net"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	"github.com/pivotal-cf/cf-redis-broker/agentapi"
	"github.com/pivotal-cf/cf-redis-broker/redis"
)

var _ = Describe("RemoteAgentClient", func() {
	var (
		server *ghttp.Server
		client *redis.RemoteAgentClient
		status int
		host   string
	)

	const (
		username = "username"
		password = "password"
	)

	BeforeEach(func() {
		status = http.StatusOK
		server = ghttp.NewServer()

		var (
			port string
			err  error
		)

		host, port, err = net.SplitHostPort(server.Addr())
		Expect(err).ToNot(HaveOccurred())

		client = redis.NewRemoteAgentClient(
			port,
			username,
			password,
			false,
		)
	})

	AfterEach(func() {
		server.Close()
	})

	Describe(".Reset", func() {
		BeforeEach(func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("DELETE", "/"),
					ghttp.VerifyBasicAuth(username, password),
					ghttp.RespondWithPtr(&status, nil),
				),
			)
		})

		It("makes a DELETE request to the host", func() {
			err := client.Reset(host)

			Expect(err).ToNot(HaveOccurred())
			Expect(server.ReceivedRequests()).To(HaveLen(1))
		})

		Context("when the DELETE request fails", func() {
			BeforeEach(func() {
				status = http.StatusInternalServerError
			})

			It("returns the error", func() {
				err := client.Reset(host)
				Expect(err).To(MatchError("Agent error: 500"))
			})
		})
	})

	Describe(".Credentials", func() {
		var expectedCredentials = redis.Credentials{
			Port:     12345,
			Password: "supersecret",
		}

		BeforeEach(func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/"),
					ghttp.VerifyBasicAuth(username, password),
					ghttp.RespondWithJSONEncodedPtr(&status, &expectedCredentials),
				),
			)
		})

		It("makes a GET request to the host", func() {
			_, err := client.Credentials(host)

			Expect(err).ToNot(HaveOccurred())
			Expect(server.ReceivedRequests()).To(HaveLen(1))
		})

		It("returns the correct credentials", func() {
			credentials, err := client.Credentials(host)
			Expect(err).ToNot(HaveOccurred())
			Expect(credentials).To(Equal(expectedCredentials))
		})

		Context("when the request fails", func() {
			BeforeEach(func() {
				status = http.StatusInternalServerError
			})

			It("returns an error", func() {
				_, err := client.Credentials(host)

				Expect(err).Should(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring("Agent error: 500")))
			})
		})
	})

	Describe(".Keycount", func() {
		var keycountResponse = agentapi.KeycountResponse{
			Keycount: 7,
		}

		BeforeEach(func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/keycount"),
					ghttp.VerifyBasicAuth(username, password),
					ghttp.RespondWithJSONEncodedPtr(&status, &keycountResponse),
				),
			)
		})

		It("makes a GET request to the host", func() {
			_, err := client.Keycount(host)

			Expect(err).ToNot(HaveOccurred())
			Expect(server.ReceivedRequests()).To(HaveLen(1))
		})

		It("returns the correct key count", func() {
			count, err := client.Keycount(host)

			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(keycountResponse.Keycount))
		})

		Context("when the request fails", func() {
			BeforeEach(func() {
				status = http.StatusInternalServerError
			})

			It("returns an error", func() {
				_, err := client.Keycount(host)

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring("Agent error: 500")))
			})
		})
	})
})
