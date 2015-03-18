package brokerintegration_test

import (
	"encoding/json"
	"net/http"

	"github.com/pivotal-cf/brokerapi"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Catalog", func() {

	It("returns HTTP 200", func() {
		code, _ := makeCatalogRequest()
		Ω(code).To(Equal(http.StatusOK))
	})

	var plans []brokerapi.ServicePlan
	var service brokerapi.Service

	Describe("Service", func() {

		BeforeEach(func() {
			_, body := makeCatalogRequest()

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
			Ω(service.ID).Should(Equal("123456abcdef"))
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
				Ω(plan.Description).Should(Equal("This plan provides a single Redis process on a shared VM, which is suitable for development and testing workloads"))
			})

			It("displays the correct metadata bullet points", func() {
				Ω(plan.Metadata.Bullets).Should(Equal([]string{
					"Each instance shares the same VM",
					"Single dedicated Redis process",
					"Suitable for development & testing workloads",
				}))
			})
		})

		Describe("Dedicated-vm plan", func() {
			var plan brokerapi.ServicePlan

			BeforeEach(func() {
				for _, p := range plans {
					if p.Name == "dedicated-vm" {
						plan = p
					}
				}
			})

			It("has the correct id from the config file", func() {
				Ω(plan.ID).Should(Equal("74E8984C-5F8C-11E4-86BE-07807B3B2589"))
			})

			It("displays the correct description", func() {
				Ω(plan.Description).Should(Equal("This plan provides a single Redis process on a dedicated VM, which is suitable for production workloads"))
			})

			It("displays the correct metadata bullet points", func() {
				Ω(plan.Metadata.Bullets).Should(Equal([]string{
					"Dedicated VM per instance",
					"Single dedicated Redis process",
					"Suitable for production workloads",
				}))
			})
		})
	})

	Context("When there are no dedicated nodes", func() {
		BeforeEach(func() {
			switchBroker("broker.yml-no-dedicated")

			_, body := makeCatalogRequest()

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

	Context("When there are dedicated nodes", func() {

		BeforeEach(func() {
			_, body := makeCatalogRequest()

			catalog := struct {
				Services []brokerapi.Service `json:"services"`
			}{}

			json.Unmarshal(body, &catalog)
			Ω(len(catalog.Services)).Should(Equal(1))

			plans = catalog.Services[0].Plans
		})

		It("shows both plans", func() {
			Ω(len(plans)).Should(Equal(2))

			planNames := []string{}
			for _, plan := range plans {
				planNames = append(planNames, plan.Name)
			}

			Ω(planNames).Should(ContainElement("shared-vm"))
			Ω(planNames).Should(ContainElement("dedicated-vm"))
		})

		Context("When the service instance limit is set to zero", func() {
			BeforeEach(func() {
				switchBroker("broker.yml-no-shared")

				_, body := makeCatalogRequest()

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

			It("Only shows the dedicated plan", func() {
				Ω(len(plans)).Should(Equal(1))

				dedicatedPlan := plans[0]
				Ω(dedicatedPlan.Name).Should(Equal("dedicated-vm"))
			})
		})
	})
})

func switchBroker(config string) {
	killProcess(brokerSession)
	safelyResetAllDirectories()
	brokerSession = buildAndLaunchBroker(config)
	Ω(serviceAvailable(brokerPort)).Should(BeTrue())
}
