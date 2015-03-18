package brokerintegration_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"code.google.com/p/go-uuid/uuid"
	redisclient "github.com/garyburd/redigo/redis"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf/cf-redis-broker/availability"
)

var _ = Describe("restarting processes", func() {
	Context("when an instance is provisioned, bound, and has data written to it", func() {
		var instanceID string
		var host string
		var port uint
		var password string
		var client redisclient.Conn

		processMonitorPath := buildExecutable("github.com/pivotal-cf/cf-redis-broker/cmd/processmonitor")

		configCommand := "CONFIG"

		BeforeEach(func() {
			monitorSession = launchProcessWithBrokerConfig(processMonitorPath, "broker.yml")

			instanceID = uuid.NewRandom().String()
			statusCode, _ := provisionInstance(instanceID, "shared")
			Ω(statusCode).To(Equal(201))

			bindingID := uuid.NewRandom().String()
			statusCode, body := bindInstance(instanceID, bindingID)
			Ω(statusCode).To(Equal(201))

			var parsedJSON map[string]interface{}
			json.Unmarshal(body, &parsedJSON)

			credentials := parsedJSON["credentials"].(map[string]interface{})
			port = uint(credentials["port"].(float64))
			host = credentials["host"].(string)
			password = credentials["password"].(string)

			client = BuildRedisClient(port, host, password)
		})

		AfterEach(func() {
			killProcess(monitorSession)
			client.Close()
			deprovisionInstance(instanceID)
		})

		It("is restarted", func() {
			_, err := client.Do("SET", "foo", "bar")
			Ω(err).ShouldNot(HaveOccurred())

			killRedisProcess(instanceID)

			Ω(serviceAvailable(port)).Should(BeTrue())

			client = BuildRedisClient(port, host, password)

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

				redisRunningCheck := serviceAvailableChecker(port)

				Eventually(redisRunningCheck, time.Second*time.Duration(1)).Should(BeTrue())

				killRedisProcess(instanceID)

				Eventually(redisRunningCheck, time.Second*time.Duration(1)).Should(BeFalse())
				Consistently(redisRunningCheck, durationForProcessMonitorToRestartInstance()).Should(BeFalse())
			})
		})

		It("recreates the log directory when the process monitor is restarted", func() {
			logDirPath, err := filepath.Abs(path.Join(brokerConfig.RedisConfiguration.InstanceLogDirectory, instanceID))
			Ω(err).ToNot(HaveOccurred())

			killProcess(monitorSession)
			killRedisProcess(instanceID)

			err = os.RemoveAll(logDirPath)
			Ω(err).NotTo(HaveOccurred())

			monitorSession = launchProcessWithBrokerConfig(processMonitorPath, "broker.yml")

			Ω(serviceAvailable(port)).Should(BeTrue())

			_, err = ioutil.ReadDir(logDirPath)
			Ω(err).NotTo(HaveOccurred())
		})

		Context("when config (e.g. maxmemory) gets updated", func() {
			BeforeEach(func() {
				killProcess(monitorSession)
				killRedisProcess(instanceID)

				monitorSession = launchProcessWithBrokerConfig(processMonitorPath, "broker.yml.updated_maxmemory")

				Ω(serviceAvailable(port)).Should(BeTrue())

				client = BuildRedisClient(port, host, password)
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
				sendUsr1ToProcessMonitor()

				killRedisProcess(instanceID)

				allowTimeForProcessMonitorToRestartInstances()
			})

			It("does not restart the instance", func() {
				address, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("localhost:%d", port))
				Ω(err).NotTo(HaveOccurred())

				err = availability.Check(address, 2*time.Second)
				Ω(err).To(HaveOccurred())
			})

			AfterEach(func() {
				deprovisionInstance(instanceID)
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

func killRedisProcess(instanceID string) {
	pidFilePath, err := filepath.Abs(path.Join(brokerConfig.RedisConfiguration.InstanceDataDirectory, instanceID, "redis-server.pid"))
	Ω(err).ToNot(HaveOccurred())

	fileContent, err := ioutil.ReadFile(pidFilePath)
	Ω(err).ToNot(HaveOccurred())

	pid, err := strconv.ParseInt(strings.TrimSpace(string(fileContent)), 10, 32)
	Ω(err).ToNot(HaveOccurred())

	process, err := os.FindProcess(int(pid))
	Ω(err).ToNot(HaveOccurred())

	err = process.Kill()
	Ω(err).ToNot(HaveOccurred())

	process.Wait()
}

func relaunchProcessMonitorWithConfig(processMonitorPath, brokerConfigName string) {
	killProcess(monitorSession)
	monitorSession = launchProcessWithBrokerConfig(processMonitorPath, brokerConfigName)
}

func sendUsr1ToProcessMonitor() {
	monitorSession.Signal(syscall.SIGUSR1)
}
