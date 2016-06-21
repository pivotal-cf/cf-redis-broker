package brokerintegration_test

import (
	"io/ioutil"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/cf-redis-broker/integration"
)

var _ = Describe("Debug", func() {
	Context("when basic auth credentials are correct", func() {
		It("returns HTTP 200", func() {
			code, _ := integration.ExecuteAuthenticatedHTTPRequest("GET", "http://localhost:3000/debug", brokerConfig.AuthConfiguration.Username, brokerConfig.AuthConfiguration.Password)
			Expect(code).To(Equal(http.StatusOK))
		})

		It("returns JSON representing the  debug information", func() {
			debugInfo := getDebugInfo()

			Expect(debugInfo.Pool.Count).To(Equal(3))
			Expect(debugInfo.Pool.Clusters).To(ContainElement([]string{"server1.127.0.0.1.xip.io"}))
			Expect(debugInfo.Pool.Clusters).To(ContainElement([]string{"server2.127.0.0.1.xip.io"}))
			Expect(debugInfo.Pool.Clusters).To(ContainElement([]string{"server3.127.0.0.1.xip.io"}))
			Expect(len(debugInfo.Pool.Clusters)).To(Equal(3))
		})

		Context("recycling instances", func() {
			var host string

			BeforeEach(func() {
				status, _ := brokerClient.ProvisionInstance("INSTANCE-1", "dedicated")
				Expect(status).To(Equal(http.StatusCreated))
				status, _ = brokerClient.ProvisionInstance("INSTANCE-2", "dedicated")
				Expect(status).To(Equal(http.StatusCreated))
				status, _ = brokerClient.ProvisionInstance("INSTANCE-3", "dedicated")
				Expect(status).To(Equal(http.StatusCreated))

				for _, cluster := range getDebugInfo().Allocated.Clusters {
					if cluster.ID == "INSTANCE-3" {
						host = cluster.Hosts[0]
					}
				}

				status, _ = brokerClient.DeprovisionInstance("INSTANCE-3")
				Expect(status).To(Equal(http.StatusOK))

				status, _ = brokerClient.ProvisionInstance("NEW-INSTANCE", "dedicated")
				Expect(status).To(Equal(http.StatusCreated))
			})

			AfterEach(func() {
				status, _ := brokerClient.DeprovisionInstance("NEW-INSTANCE")
				Expect(status).To(Equal(http.StatusOK))
				status, _ = brokerClient.DeprovisionInstance("INSTANCE-2")
				Expect(status).To(Equal(http.StatusOK))
				status, _ = brokerClient.DeprovisionInstance("INSTANCE-1")
				Expect(status).To(Equal(http.StatusOK))
			})

			It("reuses deprovisioned instance", func() {
				var newHost string
				debugInfo := getDebugInfo()

				Expect(debugInfo.Pool.Count).To(Equal(0))
				Expect(debugInfo.Pool.Clusters).To(BeEmpty())

				Expect(debugInfo.Allocated.Count).To(Equal(3))
				Expect(debugInfo.Allocated.Clusters).To(HaveLen(3))

				for _, cluster := range debugInfo.Allocated.Clusters {
					if cluster.ID == "NEW-INSTANCE" {
						newHost = cluster.Hosts[0]
					}
				}

				Expect(newHost).To(Equal(host))
			})
		})

		Context("when an instance is provisioned", func() {
			BeforeEach(func() {
				status, _ := brokerClient.ProvisionInstance("SOME-GUID", "dedicated")
				Expect(status).To(Equal(http.StatusCreated))
			})

			AfterEach(func() {
				status, _ := brokerClient.DeprovisionInstance("SOME-GUID")
				Expect(status).To(Equal(http.StatusOK))
			})

			It("removes a cluster from the Pool", func() {
				debugInfo := getDebugInfo()

				Expect(debugInfo.Pool.Count).To(Equal(2))
			})

			It("moves the cluster to allocated", func() {
				debugInfo := getDebugInfo()

				Expect(debugInfo.Allocated.Count).To(Equal(1))
				Expect(len(debugInfo.Allocated.Clusters)).To(Equal(1))

				host := debugInfo.Allocated.Clusters[0].Hosts[0]
				Expect(host).To(MatchRegexp(`server[1-3]\.127\.0\.0\.1\.xip\.io`))

				Expect(debugInfo.Pool.Clusters).NotTo(ContainElement([]string{host}))
			})

			Context("then deprovisioned", func() {
				BeforeEach(func() {
					status, _ := brokerClient.DeprovisionInstance("SOME-GUID")
					Expect(status).To(Equal(http.StatusOK))
				})

				AfterEach(func() {
					status, _ := brokerClient.ProvisionInstance("SOME-GUID", "dedicated")
					Expect(status).To(Equal(http.StatusCreated))
				})

				It("adds the cluster back to the Pool", func() {
					debugInfo := getDebugInfo()

					Expect(debugInfo.Pool.Count).To(Equal(3))
				})

				It("removes the cluster from Allocated", func() {
					debugInfo := getDebugInfo()

					Expect(debugInfo.Allocated.Count).To(Equal(0))
					Expect(debugInfo.Allocated.Clusters).To(BeEmpty())
				})
			})

			Context("when the instance is bound to", func() {
				BeforeEach(func() {
					status, _ := brokerClient.BindInstance("SOME-GUID", "foo-binding")
					Expect(status).To(Equal(http.StatusCreated))
				})

				It("returns the bindings", func() {
					debugInfo := getDebugInfo()

					bindings := debugInfo.Allocated.Clusters[0].Bindings
					Expect(len(bindings)).To(Equal(1))

					Expect(bindings[0].ID).To(Equal("foo-binding"))
				})

				Context("then unbound", func() {
					BeforeEach(func() {
						status, _ := brokerClient.UnbindInstance("SOME-GUID", "foo-binding")
						Expect(status).To(Equal(http.StatusOK))
					})

					It("returns no bindings", func() {
						debugInfo := getDebugInfo()

						bindings := debugInfo.Allocated.Clusters[0].Bindings
						Expect(len(bindings)).To(Equal(0))
					})
				})
			})

		})
	})

	Context("when basic auth credentials are incorrect", func() {
		It("returns 401 Unauthorized", func() {
			code, _ := executeHTTPRequest("GET", "http://localhost:3000/debug")
			Expect(code).To(Equal(http.StatusUnauthorized))
		})

		It("does not return the debug information", func() {
			_, bodyBytes := executeHTTPRequest("GET", "http://localhost:3000/debug")
			body := string(bodyBytes)
			Expect(body).To(Equal("Not Authorized\n"))
		})
	})
})

func executeHTTPRequest(method, uri string) (int, []byte) {
	client := new(http.Client)
	req, err := http.NewRequest(method, uri, nil)
	Expect(err).NotTo(HaveOccurred())
	resp, err := client.Do(req)
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	Expect(err).NotTo(HaveOccurred())

	Expect(err).NotTo(HaveOccurred())
	return resp.StatusCode, body
}
