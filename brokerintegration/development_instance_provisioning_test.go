package brokerintegration_test

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pborman/uuid"
	"github.com/pivotal-cf/cf-redis-broker/integration/helpers"
)

var _ = Describe("Provision shared instance", func() {
	var instanceID string
	var initialRedisProcessCount int

	BeforeEach(func() {
		instanceID = uuid.NewRandom().String()
		initialRedisProcessCount = getRedisProcessCount()
	})

	AfterEach(func() {
		Expect(getRedisProcessCount()).To(Equal(initialRedisProcessCount))
	})

	Context("when instance is created successfully", func() {
		AfterEach(func() {
			status, _ := brokerClient.DeprovisionInstance(instanceID)
			Expect(status).To(Equal(http.StatusOK))
		})

		It("returns 201", func() {
			status, _ := brokerClient.ProvisionInstance(instanceID, "shared")
			Expect(status).To(Equal(http.StatusCreated))
		})

		It("returns empty JSON", func() {
			_, body := brokerClient.ProvisionInstance(instanceID, "shared")
			Expect(body).To(MatchJSON("{}"))
		})

		It("starts a Redis instance", func() {
			status, _ := brokerClient.ProvisionInstance(instanceID, "shared")
			Expect(status).To(Equal(http.StatusCreated))
			Expect(getRedisProcessCount()).To(Equal(initialRedisProcessCount + 1))
		})

		It("writes a Redis config to the instance directory", func() {
			brokerClient.ProvisionInstance(instanceID, "shared")
			configPath := filepath.Join(brokerConfig.RedisConfiguration.InstanceDataDirectory, instanceID, "redis.conf")
			_, err := os.Stat(configPath)
			Expect(err).NotTo(HaveOccurred())
		})

		It("the redis instance logs to the right file", func() {
			var logContents string

			status, _ := brokerClient.ProvisionInstance(instanceID, "shared")
			Expect(status).To(Equal(http.StatusCreated))

			logFilePath := filepath.Join(brokerConfig.RedisConfiguration.InstanceLogDirectory, instanceID, "redis-server.log")

			for i := 0; i < 3; i++ {
				logBytes, err := ioutil.ReadFile(logFilePath)
				Expect(err).NotTo(HaveOccurred())
				logContents = string(logBytes)

				if strings.Contains(logContents, "Server started") {
					break
				}

				time.Sleep(time.Second)
			}

			Expect(logContents).To(ContainSubstring("Server started"))
		})
	})

	Context("when the service instance limit has been met", func() {
		BeforeEach(func() {
			for i := 1; i < 4; i++ {
				status, _ := brokerClient.ProvisionInstance(strconv.Itoa(i), "shared")
				Expect(status).To(Equal(http.StatusCreated))
			}
		})

		AfterEach(func() {
			for i := 1; i < 4; i++ {
				status, _ := brokerClient.DeprovisionInstance(strconv.Itoa(i))
				Expect(status).To(Equal(http.StatusOK))
			}
		})

		It("does not start a Redis instance", func() {
			brokerClient.ProvisionInstance("4", "shared")
			defer brokerClient.DeprovisionInstance("4")
			Expect(getRedisProcessCount()).To(Equal(initialRedisProcessCount + 3))
		})

		It("returns a 500", func() {
			status, _ := brokerClient.ProvisionInstance("4", "shared")
			defer brokerClient.DeprovisionInstance("4")
			Expect(status).To(Equal(http.StatusInternalServerError))
		})

		It("returns a useful error message in the correct JSON format", func() {
			_, body := brokerClient.ProvisionInstance("4", "shared")
			defer brokerClient.DeprovisionInstance("4")

			expected := `{"description":"instance limit for this service has been reached"}`
			Expect(string(body)).To(MatchJSON(expected))
		})
	})

	Context("when there is an error in instance setup", func() {
		AfterEach(func() {
			err := os.Chmod(helpers.TestDataDir, 0755)
			Expect(err).NotTo(HaveOccurred())
		})

		It("logs the error", func() {
			instanceID := "1"

			err := os.Chmod(helpers.TestDataDir, 0400)
			Expect(err).NotTo(HaveOccurred())
			status, _ := brokerClient.ProvisionInstance(instanceID, "shared")

			Expect(status).To(Equal(http.StatusInternalServerError))
			Expect(brokerSession.Buffer()).To(gbytes.Say(`"redis-broker.ensure-dirs-exist"`))
			Expect(brokerSession.Buffer()).To(gbytes.Say(
				`"error":"mkdir ` + helpers.TestDataDir + `/` + instanceID + `: permission denied"`,
			))
		})
	})

	Context("when the service instance already exists", func() {
		BeforeEach(func() {
			status, _ := brokerClient.ProvisionInstance(instanceID, "shared")
			Expect(status).To(Equal(http.StatusCreated))
		})

		AfterEach(func() {
			status, _ := brokerClient.DeprovisionInstance(instanceID)
			Expect(status).To(Equal(http.StatusOK))
		})

		It("should fail if we try to provision a second instance with the same ID", func() {
			numRedisProcessesBeforeExec := getRedisProcessCount()
			status, body := brokerClient.ProvisionInstance(instanceID, "shared")
			Expect(status).To(Equal(http.StatusConflict))

			Expect(string(body)).To(MatchJSON(`{}`))
			Expect(getRedisProcessCount()).To(Equal(numRedisProcessesBeforeExec))
		})
	})
})
