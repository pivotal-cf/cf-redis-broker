package brokerintegration_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"
	"path/filepath"
	"syscall"
	"time"

	redisclient "github.com/garyburd/redigo/redis"
	"github.com/pborman/uuid"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/cf-redis-broker/integration"
	"github.com/pivotal-cf/cf-redis-broker/integration/helpers"

	"github.com/pivotal-cf/cf-redis-broker/availability"
)

var _ = Describe("restarting processes", func() {
	Context("when an instance is provisioned, bound, and has data written to it", func() {
		var instanceID string
		var host string
		var port uint
		var password string
		var client redisclient.Conn

		processMonitorPath := helpers.BuildExecutable("github.com/pivotal-cf/cf-redis-broker/cmd/processmonitor")

		configCommand := "CONFIG"

		BeforeEach(func() {
			monitorSession = integration.LaunchProcessWithBrokerConfig(processMonitorPath, "broker.yml")

			instanceID = uuid.NewRandom().String()
			statusCode, _ := brokerClient.ProvisionInstance(instanceID, "shared")
			Ω(statusCode).To(Equal(201))

			bindingID := uuid.NewRandom().String()
			statusCode, body := brokerClient.BindInstance(instanceID, bindingID)
			Ω(statusCode).To(Equal(201))

			var parsedJSON map[string]interface{}
			json.Unmarshal(body, &parsedJSON)

			credentials := parsedJSON["credentials"].(map[string]interface{})
			port = uint(credentials["port"].(float64))
			host = credentials["host"].(string)
			password = credentials["password"].(string)

			client = helpers.BuildRedisClient(port, host, password)
		})

		AfterEach(func() {
			helpers.KillProcess(monitorSession)
			client.Close()
			brokerClient.DeprovisionInstance(instanceID)
		})

		It("is restarted", func() {
			_, err := client.Do("SET", "foo", "bar")
			Ω(err).ShouldNot(HaveOccurred())

			helpers.KillRedisProcess(instanceID, brokerConfig)

			Ω(helpers.ServiceAvailable(port)).Should(BeTrue())

			client = helpers.BuildRedisClient(port, host, password)

			value, err := redisclient.String(client.Do("GET", "foo"))
			Ω(err).ToNot(HaveOccurred())
			Ω(value).To(Equal("bar"))
		})

		Context("when there is a lock file for the instance", func() {
			It("is not restarted", func() {
				_, err := client.Do("SET", "foo", "bar")
				Ω(err).ShouldNot(HaveOccurred())

				lockFilePath := filepath.Join(brokerConfig.RedisConfiguration.InstanceDataDirectory, instanceID, "lock")
				lockFile, err := os.Create(lockFilePath)
				Ω(err).ShouldNot(HaveOccurred())
				lockFile.Close()

				Ω(helpers.ServiceAvailable(port)).Should(BeTrue())

				helpers.KillRedisProcess(instanceID, brokerConfig)

				Consistently(func() bool { return helpers.ServiceAvailable(port) }, durationForProcessMonitorToRestartInstance()).Should(BeFalse())
			})
		})

		It("recreates the log directory when the process monitor is restarted", func() {
			logDirPath, err := filepath.Abs(path.Join(brokerConfig.RedisConfiguration.InstanceLogDirectory, instanceID))
			Ω(err).ToNot(HaveOccurred())

			helpers.KillProcess(monitorSession)
			helpers.KillRedisProcess(instanceID, brokerConfig)

			err = os.RemoveAll(logDirPath)
			Ω(err).NotTo(HaveOccurred())

			monitorSession = integration.LaunchProcessWithBrokerConfig(processMonitorPath, "broker.yml")

			Ω(helpers.ServiceAvailable(port)).Should(BeTrue())

			_, err = ioutil.ReadDir(logDirPath)
			Ω(err).NotTo(HaveOccurred())
		})

		Context("when config (e.g. maxmemory) gets updated", func() {
			BeforeEach(func() {
				helpers.KillProcess(monitorSession)
				helpers.KillRedisProcess(instanceID, brokerConfig)

				monitorSession = integration.LaunchProcessWithBrokerConfig(processMonitorPath, "broker.yml.updated_maxmemory")

				Ω(helpers.ServiceAvailable(port)).Should(BeTrue())

				client = helpers.BuildRedisClient(port, host, password)
			})

			It("Has the new memory limit", func() {
				ret, err := redisclient.Values(client.Do(configCommand, "GET", "maxmemory"))
				Ω(err).NotTo(HaveOccurred())

				var configResponse struct {
					MaxMemory string `redis:"maxmemory"`
				}

				err = redisclient.ScanStruct(ret, &configResponse)
				Ω(err).NotTo(HaveOccurred())
				Ω(configResponse.MaxMemory).To(Equal("103809024")) // 99mb
			})

			AfterEach(func() {
				relaunchProcessMonitorWithConfig(processMonitorPath, "broker.yml")
			})
		})

		Context("when the processmonitor has received USR1", func() {
			BeforeEach(func() {
				monitorSession.Signal(syscall.SIGUSR1)

				helpers.KillRedisProcess(instanceID, brokerConfig)

				allowTimeForProcessMonitorToRestartInstances()
			})

			It("does not restart the instance", func() {
				address, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("localhost:%d", port))
				Ω(err).NotTo(HaveOccurred())

				err = availability.Check(address, 2*time.Second)
				Ω(err).To(HaveOccurred())
			})

			AfterEach(func() {
				brokerClient.DeprovisionInstance(instanceID)
				relaunchProcessMonitorWithConfig(processMonitorPath, "broker.yml")
			})
		})
	})
})

func durationForProcessMonitorToRestartInstance() time.Duration {
	return time.Second * time.Duration(brokerConfig.RedisConfiguration.ProcessCheckIntervalSeconds+1)
}

func allowTimeForProcessMonitorToRestartInstances() {
	time.Sleep(durationForProcessMonitorToRestartInstance())
}

func relaunchProcessMonitorWithConfig(processMonitorPath, brokerConfigName string) {
	helpers.KillProcess(monitorSession)
	monitorSession = integration.LaunchProcessWithBrokerConfig(processMonitorPath, brokerConfigName)
}
