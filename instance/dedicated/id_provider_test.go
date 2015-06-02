package dedicated_test

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
	"github.com/pivotal-cf/brokerapi/auth"
	"github.com/pivotal-cf/cf-redis-broker/instance"
	"github.com/pivotal-cf/cf-redis-broker/instance/dedicated"
	"github.com/pivotal-cf/cf-redis-broker/redisinstance"
)

var _ = Describe("dedicated.InstanceIDProvider", func() {
	Describe(".InstanceID", func() {
		var (
			brokerUsername     = "some-username"
			brokerPassword     = "some-password"
			clientUsername     string
			clientPassword     string
			server             *httptest.Server
			endpoint           string
			plan               instance.IDProvider
			expectedInstanceID = "some-instance-id"
			actualInstanceID   string
			instanceIDErr      error
			nodeIP             = "8.8.8.8"
		)

		JustBeforeEach(func() {
			plan = dedicated.InstanceIDProvider(endpoint, clientUsername, clientPassword)
			actualInstanceID, instanceIDErr = plan.InstanceID("", nodeIP)
		})

		Context("when the broker responds", func() {
			BeforeEach(func() {
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

			Context("when the node is not found", func() {
				BeforeEach(func() {
					nodeIP = "8.8.4.4"
				})

				It("returns an error", func() {
					Expect(instanceIDErr).To(HaveOccurred())
					Expect(instanceIDErr.Error()).To(ContainSubstring("404"))
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
