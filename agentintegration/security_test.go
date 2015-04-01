package agentintegration_test

import (
	"fmt"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf/cf-redis-broker/integration"
)

var _ = Describe("Agent Security", func() {
	var session *gexec.Session

	BeforeEach(func() {
		session = startAgentWithDefaultConfig()
	})

	AfterEach(func() {
		stopAgent(session)
	})

	Describe("Basic HTTP Authentication", func() {
		Context("With expected username and password", func() {
			It("returns HTTP code 200", func() {
				code, _ := integration.ExecuteAuthenticatedHTTPRequest("GET", "http://localhost:9876", "admin", "secret")
				Ω(code).To(Equal(200))
			})
		})

		Context("With unexpected username and password", func() {
			It("returns HTTP code 401", func() {
				req, err := http.NewRequest("GET", "http://localhost:9876", nil)
				Ω(err).ToNot(HaveOccurred())

				req.SetBasicAuth("admin", "badpassword")
				resp, err := http.DefaultClient.Do(req)
				Ω(err).ToNot(HaveOccurred())

				Ω(resp.StatusCode).To(Equal(401))
			})
		})
	})

	Describe("PORT Connectivity", func() {
		It("is available on the localhost only", func() {
			client := &http.Client{}
			_, err := client.Get("http://localhost:9876")
			Ω(err).ToNot(HaveOccurred())

			publicIPAddresses, err := integration.HostIP4Addresses()
			Ω(err).ToNot(HaveOccurred())

			for _, ipAddress := range publicIPAddresses {
				_, err = client.Get(fmt.Sprintf("http://%s:9876", ipAddress))
				Ω(err).To(HaveOccurred())
			}
		})
	})
})
