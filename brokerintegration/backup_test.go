package brokerintegration_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/cloudfoundry-incubator/candiedyaml"
	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
	"github.com/mitchellh/goamz/s3/s3test"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf/cf-redis-broker/backupconfig"
	"github.com/pivotal-cf/cf-redis-broker/redis/client"
	"github.com/pivotal-cf/cf-redis-broker/redisconf"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("backups", func() {
	Context("when the backup configuration is empty", func() {
		It("exits with status code 0", func() {
			backupSession := runBackupWithConfig(backupExecutablePath, filepath.Join("assets", "empty-backup.yml"))
			backupExitStatusCode := backupSession.Wait(time.Second * 10).ExitCode()
			Ω(backupExitStatusCode).Should(Equal(0))
		})
	})

	Context("when there are instances to back up", func() {
		var (
			backupConfigPath string
			backupConfig     *backupconfig.Config

			client         *s3.S3
			bucket         *s3.Bucket
			oldS3ServerURL string

			instanceIDs = []string{"foo", "bar"}
		)

		BeforeEach(func() {
			for _, instanceID := range instanceIDs {
				status, _ := brokerClient.ProvisionInstance(instanceID, "shared")
				Ω(status).To(Equal(http.StatusCreated))
				bindAndWriteTestData(instanceID)
			}

			backupConfigPath = filepath.Join("assets", "backup.yml")

			s3TestServer := startS3TestServer()
			backupConfig = loadBackupConfig(backupConfigPath)
			oldS3ServerURL = swapS3UrlInBackupConfig(backupConfig, backupConfigPath, s3TestServer.URL())
			client, bucket = configureS3ClientAndBucket(backupConfig)

			err := bucket.PutBucket("")
			Ω(err).ShouldNot(HaveOccurred())
		})

		AfterEach(func() {
			for _, instanceID := range instanceIDs {
				brokerClient.DeprovisionInstance(instanceID)
				bucket.Del(fmt.Sprintf("%s/%s", backupConfig.S3Configuration.Path, instanceID))
			}

			bucket.DelBucket()
			swapS3UrlInBackupConfig(backupConfig, backupConfigPath, oldS3ServerURL)
		})

		Context("when the backup command completes successfully", func() {
			It("exits with status code 0", func() {
				backupSession := runBackupWithConfig(backupExecutablePath, backupConfigPath)
				backupExitStatusCode := backupSession.Wait(time.Second * 10).ExitCode()
				Expect(backupExitStatusCode).To(Equal(0))
			})

			It("uploads redis instance RDB files to the correct S3 bucket", func() {
				runBackupWithConfig(backupExecutablePath, backupConfigPath).Wait(time.Second * 10)
				for _, instanceID := range instanceIDs {
					retrievedBackupBytes, err := bucket.Get(fmt.Sprintf("%s/%s", backupConfig.S3Configuration.Path, instanceID))
					Ω(err).NotTo(HaveOccurred())
					Ω(retrievedBackupBytes).To(Equal(readRdbFile(instanceID)))
				}
			})

			It("runs a background save", func() {
				instanceID := instanceIDs[0]
				confPath := filepath.Join(brokerConfig.RedisConfiguration.InstanceDataDirectory, instanceID, "redis.conf")
				lastSaveTime := getLastSaveTime(instanceID, confPath)

				runBackupWithConfig(backupExecutablePath, backupConfigPath).Wait(time.Second * 10)

				Expect(getLastSaveTime(instanceID, confPath)).To(BeNumerically(">", lastSaveTime))
			})

			It("creates the bucket if it does not exist and uploads a file for each instance", func() {
				err := bucket.DelBucket()
				Ω(err).NotTo(HaveOccurred())

				runBackupWithConfig(backupExecutablePath, backupConfigPath).Wait(time.Second * 10)

				for _, instanceID := range instanceIDs {
					retrievedBackupBytes, err := bucket.Get(fmt.Sprintf("%s/%s", backupConfig.S3Configuration.Path, instanceID))
					Ω(err).NotTo(HaveOccurred())
					Ω(retrievedBackupBytes).ShouldNot(BeEmpty())
				}
			})
		})

		Context("when the backup process is aborted", func() {
			It("exits with non-zero code", func() {
				backupSession := runBackupWithConfig(backupExecutablePath, backupConfigPath)
				backupExitStatusCode := backupSession.Kill().Wait().ExitCode()
				Ω(backupExitStatusCode).ShouldNot(Equal(0))
			})

			It("does not leave any files on s3", func() {
				runBackupWithConfig(backupExecutablePath, backupConfigPath).Kill().Wait()
				for _, instanceID := range instanceIDs {
					_, err := bucket.Get(fmt.Sprintf("%s/%s", backupConfig.S3Configuration.Path, instanceID))
					Ω(err).Should(MatchError("The specified key does not exist."))
				}
			})
		})

		Context("when an instance backup fails", func() {
			It("still backs up the other instances", func() {
				killRedisProcess(instanceIDs[0])

				backupExitStatusCode := runBackupWithConfig(backupExecutablePath, backupConfigPath).Wait(time.Second * 10).ExitCode()
				Ω(backupExitStatusCode).Should(Equal(1))

				retrievedBackupBytes, err := bucket.Get(fmt.Sprintf("%s/%s", backupConfig.S3Configuration.Path, instanceIDs[1]))
				Ω(err).NotTo(HaveOccurred())
				Ω(retrievedBackupBytes).ShouldNot(BeEmpty())
			})
		})
	})
})

func getLastSaveTime(instanceID string, configPath string) int64 {
	status, bindingBytes := brokerClient.BindInstance(instanceID, "somebindingID")
	Ω(status).To(Equal(http.StatusCreated))
	bindingResponse := map[string]interface{}{}
	json.Unmarshal(bindingBytes, &bindingResponse)
	credentials := bindingResponse["credentials"].(map[string]interface{})

	conf, err := redisconf.Load(configPath)
	Ω(err).ShouldNot(HaveOccurred())
	redisClient, err := client.Connect(
		credentials["host"].(string),
		conf,
	)
	Ω(err).ShouldNot(HaveOccurred())

	time, err := redisClient.LastRDBSaveTime()
	Ω(err).ShouldNot(HaveOccurred())

	return time
}

func bindAndWriteTestData(instanceID string) {
	status, bindingBytes := brokerClient.BindInstance(instanceID, "somebindingID")
	Ω(status).To(Equal(http.StatusCreated))
	bindingResponse := map[string]interface{}{}
	json.Unmarshal(bindingBytes, &bindingResponse)
	credentials := bindingResponse["credentials"].(map[string]interface{})
	port := uint(credentials["port"].(float64))
	redisClient := buildRedisClient(port, credentials["host"].(string), credentials["password"].(string))
	defer redisClient.Close()
	for i := 0; i < 20; i++ {
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

func swapS3UrlInBackupConfig(config *backupconfig.Config, path, newEndpointURL string) string {
	oldEndpointURL := config.S3Configuration.EndpointUrl
	config.S3Configuration.EndpointUrl = newEndpointURL

	configFile, err := os.Create(path)
	Ω(err).ToNot(HaveOccurred())
	encoder := candiedyaml.NewEncoder(configFile)
	err = encoder.Encode(config)
	Ω(err).ToNot(HaveOccurred())

	return oldEndpointURL
}

func runBackupWithConfig(executablePath, configPath string) *gexec.Session {
	cmd := exec.Command(executablePath)
	cmd.Stdout = GinkgoWriter
	cmd.Stderr = GinkgoWriter
	cmd.Env = append(cmd.Env, "BACKUP_CONFIG_PATH="+configPath)
	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	return session
}

func configureS3ClientAndBucket(backupConfig *backupconfig.Config) (*s3.S3, *s3.Bucket) {
	client := s3.New(aws.Auth{
		AccessKey: backupConfig.S3Configuration.AccessKeyId,
		SecretKey: backupConfig.S3Configuration.SecretAccessKey,
	}, aws.Region{
		Name:                 backupConfig.S3Configuration.Region,
		S3Endpoint:           backupConfig.S3Configuration.EndpointUrl,
		S3LocationConstraint: true,
	})
	return client, client.Bucket(backupConfig.S3Configuration.BucketName)
}

func startS3TestServer() *s3test.Server {
	s3testServer, err := s3test.NewServer(&s3test.Config{
		Send409Conflict: true,
	})
	Ω(err).ToNot(HaveOccurred())
	return s3testServer
}

func loadBackupConfig(path string) *backupconfig.Config {
	backupConfig, err := backupconfig.Load(path)
	Expect(err).NotTo(HaveOccurred())
	return backupConfig
}
