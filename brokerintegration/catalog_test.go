package brokerintegration_test

import (
	"encoding/json"
	"net/http"

	"github.com/pivotal-cf/brokerapi/v10"
	"github.com/pivotal-cf/cf-redis-broker/integration"
	"github.com/pivotal-cf/cf-redis-broker/integration/helpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Catalog", func() {

	It("returns HTTP 200", func() {
		code, _ := brokerClient.MakeCatalogRequest()
		Ω(code).To(Equal(http.StatusOK))
	})

	var plans []brokerapi.ServicePlan
	var service brokerapi.Service

	Describe("Service", func() {

		BeforeEach(func() {
			_, body := brokerClient.MakeCatalogRequest()

			catalog := struct {
				Services []brokerapi.Service `json:"services"`
			}{}

			json.Unmarshal(body, &catalog)
			Ω(len(catalog.Services)).Should(Equal(1))

			service = catalog.Services[0]
			plans = service.Plans
		})

		It("displays the correct service name and id", func() {
			Ω(service.Name).Should(Equal("my-redis"))
			Ω(service.ID).Should(Equal("7C257149-B342-4BFC-AE51-C195F376D669"))
		})

		It("displays the correct documentation URL", func() {
			Ω(service.Metadata.DocumentationUrl).Should(Equal("http://docs.pivotal.io/p1-services/Redis.html"))
		})

		It("displays the correct support URL", func() {
			Ω(service.Metadata.SupportUrl).Should(Equal("http://support.pivotal.io"))
		})

		It("displays the description", func() {
			Ω(service.Description).Should(Equal("Redis service to provide a key-value store"))
		})

		Describe("Shared-vm plan", func() {
			var plan brokerapi.ServicePlan

			BeforeEach(func() {
				for _, p := range plans {
					if p.Name == "shared-vm" {
						plan = p
					}
				}
			})

			It("has the correct id from the config file", func() {
				Ω(plan.ID).Should(Equal("C210CA06-E7E5-4F5D-A5AA-7A2C51CC290E"))
			})

			It("displays the correct description", func() {
				Ω(plan.Description).Should(Equal("This plan provides a Redis server on a shared VM configured for data persistence."))
			})

			It("displays the correct metadata bullet points", func() {
				Ω(plan.Metadata.Bullets).Should(Equal([]string{
					"Each instance shares the same VM",
					"Single dedicated Redis process",
					"Suitable for development & testing workloads",
				}))
			})
		})
	})

	Context("When there are no dedicated nodes", func() {
		BeforeEach(func() {
			switchBroker("broker.yml-no-dedicated")

			_, body := brokerClient.MakeCatalogRequest()

			catalog := struct {
				Services []brokerapi.Service `json:"services"`
			}{}

			json.Unmarshal(body, &catalog)
			Ω(len(catalog.Services)).Should(Equal(1))

			plans = catalog.Services[0].Plans
		})

		AfterEach(func() {
			switchBroker("broker.yml")
		})

		It("only shows the shared plan", func() {
			Ω(len(plans)).Should(Equal(1))

			sharedPlan := plans[0]
			Ω(sharedPlan.Name).Should(Equal("shared-vm"))
		})
	})
})

func switchBroker(config string) {
	helpers.KillProcess(brokerSession)
	helpers.ResetTestDirs()
	brokerSession = integration.LaunchProcessWithBrokerConfig(brokerExecutablePath, config)
	Ω(helpers.ServiceAvailable(brokerPort)).Should(BeTrue())
}
