package brokerintegration_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf/cf-redis-broker/redis/client"
	"github.com/pivotal-cf/cf-redis-broker/redisconf"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("backups", func() {
	var (
		instanceIDs = []string{"foo", "bar"}
		keysToWrite int

		brokerConfigPath string
		client           *s3.S3
		bucket           *s3.Bucket
		s3Path           string

		backupExitStatusCode int

		backupSession *gexec.Session
	)

	BeforeEach(func() {
		keysToWrite = 1
		brokerConfigPath = "broker.yml"
		backupConfig := brokerConfig.RedisConfiguration.BackupConfiguration
		s3Path = backupConfig.Path
		region := aws.Region{
			Name:                 backupConfig.S3Region,
			S3Endpoint:           backupConfig.EndpointUrl,
			S3LocationConstraint: true,
		}
		client = s3.New(aws.Auth{
			AccessKey: backupConfig.AccessKeyId,
			SecretKey: backupConfig.SecretAccessKey,
		}, region)
		bucket = client.Bucket(backupConfig.BucketName)
	})

	AfterEach(func() {
		bucket.DelBucket()
	})

	Context("when there is a dump.rdb to back up", func() {
		var lastSaveTimes map[string]int64

		JustBeforeEach(func() {
			lastSaveTimes = map[string]int64{}

			for _, instanceID := range instanceIDs {
				status, _ := provisionInstance(instanceID, "shared")
				Ω(status).To(Equal(http.StatusCreated))
				bindAndWriteTestData(instanceID, keysToWrite)

				confPath := filepath.Join(brokerConfig.RedisConfiguration.InstanceDataDirectory, instanceID, "redis.conf")
				lastSaveTimes[instanceID] = getLastSaveTime(instanceID, confPath)
			}
			backupSession = launchProcessWithBrokerConfig(backupExecutablePath, brokerConfigPath)
		})

		AfterEach(func() {
			for _, instanceID := range instanceIDs {
				status, _ := deprovisionInstance(instanceID)
				Ω(status).To(Equal(http.StatusOK))
				bucket.Del(fmt.Sprintf("%s/%s", s3Path, instanceID))
			}
		})

		Context("when the backup has completed", func() {

			JustBeforeEach(func() {
				backupExitStatusCode = backupSession.Wait(time.Second * 10).ExitCode()
				Eventually(backupSession).Should(gexec.Exit())
			})

			Context("when the bucket exists", func() {
				BeforeEach(func() {
					err := bucket.PutBucket("")
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("uploads redis instance RDB files to the correct S3 bucket", func() {
					for _, instanceID := range instanceIDs {
						retrievedBackupBytes, err := bucket.Get(fmt.Sprintf("%s/%s", s3Path, instanceID))
						Ω(err).NotTo(HaveOccurred())
						Ω(retrievedBackupBytes).To(Equal(readRdbFile(instanceID)))
					}
				})

				It("runs a background save", func() {
					// check that every instance has been saved
					// since the backup started
					checkLatestSaveTime := func() bool {
						for _, instanceID := range instanceIDs {
							confPath := filepath.Join(brokerConfig.RedisConfiguration.InstanceDataDirectory, instanceID, "redis.conf")
							if getLastSaveTime(instanceID, confPath) <= lastSaveTimes[instanceID] {
								return false
							}
						}
						return true
					}
					Eventually(checkLatestSaveTime).Should(BeTrue())
				})
			})

			Context("when the bucket does not exist", func() {
				It("creates the bucket and uploads a file for each instance", func() {
					for _, instanceID := range instanceIDs {
						retrievedBackupBytes, err := bucket.Get(fmt.Sprintf("%s/%s", s3Path, instanceID))
						Ω(err).NotTo(HaveOccurred())
						Ω(retrievedBackupBytes).ShouldNot(BeEmpty())
					}
				})
			})

			Context("when the backup configuration is empty", func() {
				BeforeEach(func() {
					brokerConfigPath = "broker.yml-no-backup"
				})

				It("exits with status code 0", func() {
					Ω(backupExitStatusCode).Should(Equal(0))
				})

				It("does not create an empty bucket", func() {
					resp, err := client.ListBuckets()
					Ω(err).ShouldNot(HaveOccurred())
					Ω(resp.Buckets).Should(BeNil())
				})
			})
		})

		Context("when the backup process is aborted", func() {

			BeforeEach(func() {
				keysToWrite = 20
			})

			JustBeforeEach(func() {
				backupExitStatusCode = backupSession.Kill().Wait().ExitCode()
				Eventually(backupSession).Should(gexec.Exit())
			})

			Context("when the bucket exists", func() {
				BeforeEach(func() {
					err := bucket.PutBucket("")
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("exits with non-zero code", func() {
					Ω(backupExitStatusCode).ShouldNot(Equal(0))
				})

				It("does not leave any files on s3", func() {
					for _, instanceID := range instanceIDs {
						_, err := bucket.Get(fmt.Sprintf("%s/%s", s3Path, instanceID))
						Ω(err).Should(MatchError("The specified key does not exist."))
					}
				})
			})
		})
	})

	Context("when backing up multiple instances", func() {
		Context("when an error happens with one of the intances", func() {
			It("still backups the other intances", func() {
				status, _ := provisionInstance("A", "shared")
				Ω(status).To(Equal(http.StatusCreated))

				status, _ = provisionInstance("B", "shared")
				Ω(status).To(Equal(http.StatusCreated))

				// killing redis will cause backup "A" to fail
				killRedisProcess("A")

				bindAndWriteTestData("B", keysToWrite)
				backupSession = launchProcessWithBrokerConfig(backupExecutablePath, brokerConfigPath)
				backupExitStatusCode = backupSession.Wait(time.Second * 10).ExitCode()

				// backup should fail
				Ω(backupExitStatusCode).Should(Equal(1))

				// but B should still be backed up.
				retrievedBackupBytes, err := bucket.Get(fmt.Sprintf("%s/%s", s3Path, "B"))
				Ω(err).NotTo(HaveOccurred())
				Ω(retrievedBackupBytes).ShouldNot(BeEmpty())

				status, _ = deprovisionInstance("B")
				Ω(status).To(Equal(http.StatusOK))

				deprovisionInstance("A")

				os.RemoveAll(brokerConfig.RedisConfiguration.InstanceDataDirectory)

				bucket.Del(fmt.Sprintf("%s/%s", s3Path, "B"))
			})
		})
	})
})

func getLastSaveTime(instanceID string, configPath string) int64 {
	status, bindingBytes := bindInstance(instanceID, "somebindingID")
	Ω(status).To(Equal(http.StatusCreated))
	bindingResponse := map[string]interface{}{}
	json.Unmarshal(bindingBytes, &bindingResponse)
	credentials := bindingResponse["credentials"].(map[string]interface{})
	port := uint(credentials["port"].(float64))

	conf, err := redisconf.Load(configPath)
	Ω(err).ShouldNot(HaveOccurred())
	redisClient, err := client.Connect(
		credentials["host"].(string),
		port,
		credentials["password"].(string),
		conf,
	)
	Ω(err).ShouldNot(HaveOccurred())

	time, err := redisClient.LastRDBSaveTime()
	Ω(err).ShouldNot(HaveOccurred())

	return time
}

func bindAndWriteTestData(instanceID string, size int) {
	status, bindingBytes := bindInstance(instanceID, "somebindingID")
	Ω(status).To(Equal(http.StatusCreated))
	bindingResponse := map[string]interface{}{}
	json.Unmarshal(bindingBytes, &bindingResponse)
	credentials := bindingResponse["credentials"].(map[string]interface{})
	port := uint(credentials["port"].(float64))
	redisClient := BuildRedisClient(port, credentials["host"].(string), credentials["password"].(string))
	defer redisClient.Close()
	for i := 0; i < size; i++ {
		_, err := redisClient.Do("SET", fmt.Sprintf("foo%d", i), fmt.Sprintf("bar%d", i))
		Ω(err).ToNot(HaveOccurred())
	}
	_, err := redisClient.Do("SAVE")
	Ω(err).ToNot(HaveOccurred())
}

func readRdbFile(instanceID string) []byte {
	instanceDataDir := brokerConfig.RedisConfiguration.InstanceDataDirectory
	pathToRdbFile := filepath.Join(instanceDataDir, instanceID, "db", "dump.rdb")
	originalRdbBytes, err := ioutil.ReadFile(pathToRdbFile)
	Ω(err).ToNot(HaveOccurred())
	return originalRdbBytes
}
