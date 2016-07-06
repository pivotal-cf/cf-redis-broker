package brokerintegration_test

import (
	"fmt"
	"sort"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/pborman/uuid"
	"github.com/pivotal-cf/cf-redis-broker/integration"
	"github.com/pivotal-cf/cf-redis-broker/integration/helpers"
)

var sourcePath = "github.com/pivotal-cf/cf-redis-broker/cmd/processmonitor"

var _ = Describe("processmonitor cmd", func() {
	Describe("Log output", func() {
		var monitorSession *gexec.Session
		processMonitorPath := helpers.BuildExecutable(sourcePath)

		Context("When there are no Redis instances provisioned", func() {
			BeforeEach(func() {
				monitorSession = integration.LaunchProcessWithBrokerConfig(processMonitorPath, "broker.yml")
			})

			AfterEach(func() {
				helpers.KillProcess(monitorSession)
			})

			It("logs 0 instances found", func() {
				Eventually(monitorSession.Buffer()).Should(gbytes.Say("0 shared Redis instances found"))
			})
		})

		Context("When there is a single Redis instance provisioned", func() {
			instanceUuid := uuid.NewRandom().String()

			BeforeEach(func() {
				statusCode, _ := brokerClient.ProvisionInstance(instanceUuid, "shared")
				Expect(statusCode).To(Equal(201))
				monitorSession = integration.LaunchProcessWithBrokerConfig(processMonitorPath, "broker.yml")
			})

			AfterEach(func() {
				helpers.KillProcess(monitorSession)
				statusCode, _ := brokerClient.DeprovisionInstance(instanceUuid)
				Expect(statusCode).To(Equal(200))
			})

			It("logs 1 instance found", func() {
				Eventually(monitorSession.Buffer()).Should(
					gbytes.Say("1 shared Redis instance found"),
				)
			})

			It("logs the instance ID", func() {
				Eventually(monitorSession.Buffer()).Should(
					gbytes.Say(fmt.Sprintf("Found shared instance: %s", instanceUuid)),
				)
			})
		})

		Context("When there are multiple Redis instances provisioned", func() {
			instanceUuids := []string{
				uuid.NewRandom().String(),
				uuid.NewRandom().String(),
			}
			sort.Strings(instanceUuids)

			BeforeEach(func() {
				for _, uuid := range instanceUuids {
					statusCode, _ := brokerClient.ProvisionInstance(uuid, "shared")
					Expect(statusCode).To(Equal(201))
				}

				monitorSession = integration.LaunchProcessWithBrokerConfig(processMonitorPath, "broker.yml")
			})

			AfterEach(func() {
				helpers.KillProcess(monitorSession)
				for _, uuid := range instanceUuids {
					statusCode, _ := brokerClient.DeprovisionInstance(uuid)
					Expect(statusCode).To(Equal(200))
				}
			})

			It("logs 2 instances found", func() {
				Eventually(monitorSession.Buffer()).Should(
					gbytes.Say("2 shared Redis instances found"),
				)
			})

			It("logs the 2 instance IDs", func() {
				Eventually(monitorSession.Buffer()).Should(
					gbytes.Say(fmt.Sprintf("Found shared instance: %s", instanceUuids[0])),
				)

				Eventually(monitorSession.Buffer()).Should(
					gbytes.Say(fmt.Sprintf("Found shared instance: %s", instanceUuids[1])),
				)
			})
		})
	})
})
