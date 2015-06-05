package id_test

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-cf/brokerapi/auth"
	"github.com/pivotal-cf/cf-redis-broker/instance/id"
	"github.com/pivotal-cf/cf-redis-broker/redisinstance"
	"github.com/pivotal-golang/lager"
	. "github.com/st3v/glager"
)

var _ = Describe("InstanceIDLocator", func() {
	Describe(".InstanceID", func() {
		var (
			brokerUsername     = "some-username"
			brokerPassword     = "some-password"
			clientUsername     string
			clientPassword     string
			server             *httptest.Server
			endpoint           string
			expectedURL        string
			idLocator          id.InstanceIDLocator
			expectedInstanceID = "some-instance-id"
			actualInstanceID   string
			instanceIDErr      error
			nodeIP             string
			log                *gbytes.Buffer
		)

		JustBeforeEach(func() {
			log = gbytes.NewBuffer()
			logger := lager.NewLogger("provider")
			logger.RegisterSink(lager.NewWriterSink(log, lager.DEBUG))

			idLocator = id.DedicatedInstanceIDLocator(endpoint, clientUsername, clientPassword, logger)
			actualInstanceID, instanceIDErr = idLocator.LocateID("", nodeIP)

			expectedURL = fmt.Sprintf("%s?host=%s", endpoint, nodeIP)
		})

		Context("when the broker responds", func() {
			BeforeEach(func() {
				nodeIP = "8.8.8.8"

				authWrapper := auth.NewWrapper(brokerUsername, brokerPassword)

				instanceHandler := authWrapper.WrapFunc(redisinstance.NewHandler(&instanceFinder{
					InstanceID: expectedInstanceID,
					InstanceIP: nodeIP,
				}))

				server = httptest.NewServer(instanceHandler)
				endpoint = server.URL

				clientUsername = brokerUsername
				clientPassword = brokerPassword
			})

			AfterEach(func() {
				server.Close()
			})

			It("does not return an error", func() {
				Expect(instanceIDErr).ToNot(HaveOccurred())
			})

			It("returns the instance ID", func() {
				Expect(actualInstanceID).To(Equal(expectedInstanceID))
			})

			It("provides logging", func() {
				Expect(log).To(ContainSequence(
					Info(
						Action("provider.dedicated-instance-id"),
						Data("event", "starting", "node_ip", nodeIP),
					),
					Info(
						Action("provider.broker-request"),
						Data("event", "starting", "url", expectedURL),
					),
					Info(
						Action("provider.broker-request"),
						Data("event", "done", "url", expectedURL),
					),
					Info(
						Action("provider.dedicated-instance-id"),
						Data("event", "done", "node_ip", nodeIP, "instance_id", expectedInstanceID),
					),
				))
			})

			Context("when the node is not found", func() {
				BeforeEach(func() {
					nodeIP = "8.8.4.4"
				})

				It("returns an error", func() {
					Expect(instanceIDErr).To(HaveOccurred())
					Expect(instanceIDErr.Error()).To(ContainSubstring("404"))
				})

				It("logs the error", func() {
					Expect(log).To(ContainSequence(
						Error(
							instanceIDErr,
							Action("provider.check-response-status"),
							Data("event", "failed", "url", expectedURL),
						),
					))
				})
			})

			Context("when the broker credentials are wrong", func() {
				BeforeEach(func() {
					clientPassword = "incorrect"
				})

				It("returns an error", func() {
					Expect(instanceIDErr).To(HaveOccurred())
					Expect(instanceIDErr.Error()).To(ContainSubstring("401"))
				})

				It("logs the error", func() {
					Expect(log).To(ContainSequence(
						Error(
							instanceIDErr,
							Action("provider.check-response-status"),
							Data("event", "failed", "url", expectedURL),
						),
					))
				})
			})
		})

		Context("when the broker does not respond", func() {
			BeforeEach(func() {
				endpoint = fmt.Sprintf("http://localhost:%s/non-existing/", freePort())
			})

			It("returns an error", func() {
				Expect(instanceIDErr).To(HaveOccurred())
				Expect(instanceIDErr).To(BeAssignableToTypeOf(&url.Error{}))
			})

			It("logs the error", func() {
				Expect(log).To(ContainSequence(
					Error(
						instanceIDErr,
						Action("provider.broker-request"),
						Data("event", "failed", "url", expectedURL),
					),
				))
			})
		})

		Context("when the broker response is invalid", func() {
			BeforeEach(func() {
				authWrapper := auth.NewWrapper(brokerUsername, brokerPassword)

				instanceHandler := authWrapper.WrapFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Add("Content-Type", "application/json")
					fmt.Fprint(w, "foo")
				})

				server = httptest.NewServer(instanceHandler)
				endpoint = server.URL

				clientUsername = brokerUsername
				clientPassword = brokerPassword
			})

			AfterEach(func() {
				server.Close()
			})

			It("returns an error", func() {
				Expect(instanceIDErr).To(HaveOccurred())
				Expect(instanceIDErr).To(BeAssignableToTypeOf(&json.SyntaxError{}))
			})

			It("logs the error", func() {
				Expect(log).To(ContainSequence(
					Error(
						nil,
						Action("provider.unmarshal-response-body"),
						Data("event", "failed"),
					),
				))
			})
		})
	})
})

type instanceFinder struct {
	InstanceID string
	InstanceIP string
}

func (f *instanceFinder) IDForHost(hostIP string) string {
	if f.InstanceIP == hostIP {
		return f.InstanceID
	}

	return ""
}

func freePort() string {
	l, _ := net.Listen("tcp", ":0")
	defer l.Close()
	parts := strings.Split(l.Addr().String(), ":")
	return parts[len(parts)-1]
}
