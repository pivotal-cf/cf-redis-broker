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
			Ω(code).To(Equal(http.StatusOK))
		})

		It("returns JSON representing the  debug information", func() {
			debugInfo := getDebugInfo()

			Ω(debugInfo.Pool.Count).Should(Equal(3))
			Ω(debugInfo.Pool.Clusters).Should(ContainElement([]string{"server1.127.0.0.1.xip.io"}))
			Ω(debugInfo.Pool.Clusters).Should(ContainElement([]string{"server2.127.0.0.1.xip.io"}))
			Ω(debugInfo.Pool.Clusters).Should(ContainElement([]string{"server3.127.0.0.1.xip.io"}))
			Ω(len(debugInfo.Pool.Clusters)).Should(Equal(3))
		})

		Context("recycling instances", func() {
			var host string

			BeforeEach(func() {
				brokerClient.ProvisionInstance("INSTANCE-1", "dedicated")
				brokerClient.ProvisionInstance("INSTANCE-2", "dedicated")
				brokerClient.ProvisionInstance("INSTANCE-3", "dedicated")

				for _, cluster := range getDebugInfo().Allocated.Clusters {
					if cluster.ID == "INSTANCE-3" {
						host = cluster.Hosts[0]
					}
				}

				brokerClient.DeprovisionInstance("INSTANCE-3")

				brokerClient.ProvisionInstance("NEW-INSTANCE", "dedicated")
			})

			AfterEach(func() {
				brokerClient.DeprovisionInstance("NEW-INSTANCE")
				brokerClient.DeprovisionInstance("INSTANCE-2")
				brokerClient.DeprovisionInstance("INSTANCE-1")
			})

			It("reuses deprovisioned instance", func() {
				var newHost string
				debugInfo := getDebugInfo()

				Ω(debugInfo.Pool.Count).Should(Equal(0))
				Ω(debugInfo.Pool.Clusters).Should(BeEmpty())

				Ω(debugInfo.Allocated.Count).Should(Equal(3))
				Ω(debugInfo.Allocated.Clusters).Should(HaveLen(3))

				for _, cluster := range debugInfo.Allocated.Clusters {
					if cluster.ID == "NEW-INSTANCE" {
						newHost = cluster.Hosts[0]
					}
				}

				Ω(newHost).To(Equal(host))
			})
		})

		Context("when an instance is provisioned", func() {
			BeforeEach(func() {
				brokerClient.ProvisionInstance("SOME-GUID", "dedicated")
			})

			AfterEach(func() {
				status, _ := brokerClient.DeprovisionInstance("SOME-GUID")
				Ω(status).Should(Equal(200))
			})

			It("removes a cluster from the Pool", func() {
				debugInfo := getDebugInfo()

				Ω(debugInfo.Pool.Count).Should(Equal(2))
			})

			It("moves the cluster to allocated", func() {
				debugInfo := getDebugInfo()

				Ω(debugInfo.Allocated.Count).Should(Equal(1))
				Ω(len(debugInfo.Allocated.Clusters)).Should(Equal(1))

				host := debugInfo.Allocated.Clusters[0].Hosts[0]
				Ω(host).Should(MatchRegexp("server[1-3]\\.127\\.0\\.0\\.1\\.xip\\.io"))

				Ω(debugInfo.Pool.Clusters).ShouldNot(ContainElement([]string{host}))
			})

			Context("then deprovisioned", func() {
				BeforeEach(func() {
					status, _ := brokerClient.DeprovisionInstance("SOME-GUID")
					Ω(status).Should(Equal(200))
				})

				AfterEach(func() {
					brokerClient.ProvisionInstance("SOME-GUID", "dedicated")
				})

				It("adds the cluster back to the Pool", func() {
					debugInfo := getDebugInfo()

					Ω(debugInfo.Pool.Count).Should(Equal(3))
				})

				It("removes the cluster from Allocated", func() {
					debugInfo := getDebugInfo()

					Ω(debugInfo.Allocated.Count).Should(Equal(0))
					Ω(debugInfo.Allocated.Clusters).Should(BeEmpty())
				})
			})

			Context("when the instance is bound to", func() {
				BeforeEach(func() {
					status, _ := brokerClient.BindInstance("SOME-GUID", "foo-binding")
					Ω(status).Should(Equal(http.StatusCreated))
				})

				It("returns the bindings", func() {
					debugInfo := getDebugInfo()

					bindings := debugInfo.Allocated.Clusters[0].Bindings
					Ω(len(bindings)).Should(Equal(1))

					Ω(bindings[0].ID).Should(Equal("foo-binding"))
				})

				Context("then unbound", func() {
					BeforeEach(func() {
						status, _ := brokerClient.UnbindInstance("SOME-GUID", "foo-binding")
						Ω(status).Should(Equal(http.StatusOK))
					})

					It("returns no bindings", func() {
						debugInfo := getDebugInfo()

						bindings := debugInfo.Allocated.Clusters[0].Bindings
						Ω(len(bindings)).Should(Equal(0))
					})
				})
			})

		})
	})

	Context("when basic auth credentials are incorrect", func() {
		It("returns 401 Unauthorized", func() {
			code, _ := executeHTTPRequest("GET", "http://localhost:3000/debug")
			Ω(code).Should(Equal(http.StatusUnauthorized))
		})

		It("does not return the debug information", func() {
			_, bodyBytes := executeHTTPRequest("GET", "http://localhost:3000/debug")
			body := string(bodyBytes)
			Ω(body).Should(Equal("Unauthorized"))
		})
	})
})

func executeHTTPRequest(method string, uri string) (int, []byte) {
	client := &http.Client{}
	req, err := http.NewRequest(method, uri, nil)
	Ω(err).ToNot(HaveOccurred())
	resp, err := client.Do(req)
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	Ω(err).ToNot(HaveOccurred())

	Ω(err).ToNot(HaveOccurred())
	return resp.StatusCode, body
}
