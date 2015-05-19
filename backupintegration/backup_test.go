package backupintegration_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"code.google.com/p/go-uuid/uuid"

	"github.com/cloudfoundry-incubator/candiedyaml"
	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf-experimental/goamz/s3/s3test"
	"github.com/pivotal-cf/cf-redis-broker/backupconfig"
	"github.com/pivotal-cf/cf-redis-broker/integration"
	"github.com/pivotal-cf/cf-redis-broker/integration/helpers"
	"github.com/pivotal-cf/cf-redis-broker/redis/client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("backups", func() {
	Context("when the backup configuration is empty", func() {
		It("exits with status code 0", func() {
			backupSession, _ := runBackupWithConfig(backupExecutablePath, filepath.Join("assets", "empty-backup.yml"))
			backupExitStatusCode := backupSession.Wait(time.Second * 10).ExitCode()
			Ω(backupExitStatusCode).Should(Equal(0))
		})
	})

	Context("when the configuration is not empty", func() {
		var (
			backupConfigPath string
			backupConfig     *backupconfig.Config

			client         *s3.S3
			bucket         *s3.Bucket
			oldS3ServerURL string
			oldCliPath     string
			s3TestServer   *s3test.Server
		)

		BeforeEach(func() {
			s3TestServer = startS3TestServer()
		})

		JustBeforeEach(func() {
			err := bucket.PutBucket("")
			Ω(err).ShouldNot(HaveOccurred())
		})

		AfterEach(func() {
			swapS3ValuesInBackupConfig(backupConfig, backupConfigPath, oldS3ServerURL, oldCliPath)
		})

		Context("when its a dedicated instance to back up", func() {
			var redisRunner *integration.RedisRunner
			var confPath string
			var instanceID string

			BeforeEach(func() {
				backupConfigPath = filepath.Join("assets", "backup-dedicated.yml")
				backupConfig = loadBackupConfig(backupConfigPath)

				oldS3ServerURL = backupConfig.S3Configuration.EndpointUrl
				oldCliPath = backupConfig.AwsCLIPath
				backupConfig.S3Configuration.EndpointUrl = s3TestServer.URL()
				backupConfig.AwsCLIPath = awsCliPath
				writeConfig(backupConfig, backupConfigPath)

				client, bucket = configureS3ClientAndBucket(backupConfig)

				instanceID = uuid.NewRandom().String()
				confPath = filepath.Join(brokerConfig.RedisConfiguration.InstanceDataDirectory, "redis.conf")
				redisConfContents, err := ioutil.ReadFile(helpers.AssetPath("redis-dedicated.conf"))
				Ω(err).ShouldNot(HaveOccurred())

				ioutil.WriteFile(confPath, redisConfContents, 0777)
				redisRunner = &integration.RedisRunner{}
				redisRunner.Start([]string{confPath, "--port", "6480"})

				status, _ := brokerClient.ProvisionInstance(instanceID, "dedicated")
				Ω(status).To(Equal(http.StatusCreated))
				bindAndWriteTestData(instanceID)
			})

			AfterEach(func() {
				brokerClient.DeprovisionInstance(instanceID)
				redisRunner.Stop()
			})

			It("causes redis to SAVE/BGSAVE before backing up", func() {
				lastSaveTime := getLastSaveTime(instanceID, confPath)

				backupSession, _ := runBackupWithConfig(backupExecutablePath, backupConfigPath)
				backupExitStatusCode := backupSession.ExitCode()
				Expect(backupExitStatusCode).To(Equal(0))

				Expect(getLastSaveTime(instanceID, confPath)).To(BeNumerically(">", lastSaveTime))
			})

			It("uploads redis instance RDB file to the correct S3 bucket", func() {
				timestamp := getCurrentTimestamp()
				_, cliCommand := runBackupWithConfig(backupExecutablePath, backupConfigPath)
				Expect(cliCommand).To(MatchRegexp(cliParams(*backupConfig, bucket.Name, timestamp, instanceID, "dedicated-vm", s3TestServer.URL())))
			})

			It("creates the bucket if it does not exist", func() {
				err := bucket.DelBucket()
				Ω(err).NotTo(HaveOccurred())

				buckets, err := client.ListBuckets()
				Expect(err).ToNot(HaveOccurred())
				Expect(buckets.Buckets).To(HaveLen(0))

				runBackupWithConfig(backupExecutablePath, backupConfigPath)

				buckets, err = client.ListBuckets()
				Expect(err).ToNot(HaveOccurred())
				Expect(buckets.Buckets).To(HaveLen(1))
				Expect(buckets.Buckets[0].Name).To(Equal(backupConfig.S3Configuration.BucketName))
			})

			Context("When broker is not responding", func() {
				It("returns non-zero exit code", func() {
					backupSession, _ := runBackupWithConfig(backupExecutablePath, helpers.AssetPath("backup-dedicated-with-wrong-broker.yml"))
					backupExitStatusCode := backupSession.ExitCode()

					Ω(backupExitStatusCode).ShouldNot(Equal(0))
				})
			})

			Context("when an instance backup fails", func() {
				It("returns non-zero exit code", func() {
					redisRunner.Stop()
					backupSession, _ := runBackupWithConfig(backupExecutablePath, backupConfigPath)
					backupExitStatusCode := backupSession.ExitCode()

					Ω(backupExitStatusCode).ShouldNot(Equal(0))
				})
			})
		})

		Context("when there are shared-vm instances to back up", func() {
			var instanceIDs = []string{"foo", "bar"}

			BeforeEach(func() {
				backupConfigPath = filepath.Join("assets", "backup-shared.yml")
				backupConfig = loadBackupConfig(backupConfigPath)

				oldS3ServerURL = backupConfig.S3Configuration.EndpointUrl
				oldCliPath = backupConfig.AwsCLIPath
				backupConfig.S3Configuration.EndpointUrl = s3TestServer.URL()
				backupConfig.AwsCLIPath = awsCliPath
				writeConfig(backupConfig, backupConfigPath)

				client, bucket = configureS3ClientAndBucket(backupConfig)

				for _, instanceID := range instanceIDs {
					status, _ := brokerClient.ProvisionInstance(instanceID, "shared")
					Expect(status).To(Equal(http.StatusCreated))
					bindAndWriteTestData(instanceID)
				}
			})

			AfterEach(func() {
				for _, instanceID := range instanceIDs {
					brokerClient.DeprovisionInstance(instanceID)
					bucket.Del(fmt.Sprintf("%s/%s", backupConfig.S3Configuration.Path, instanceID))
				}

				bucket.DelBucket()
				swapS3ValuesInBackupConfig(backupConfig, backupConfigPath, oldS3ServerURL, oldCliPath)
			})

			Context("when the backup command completes successfully", func() {
				It("exits with status code 0", func() {
					backupSession, _ := runBackupWithConfig(backupExecutablePath, backupConfigPath)
					backupExitStatusCode := backupSession.Wait(time.Second * 10).ExitCode()
					Expect(backupExitStatusCode).To(Equal(0))
				})

				It("uploads redis instance RDB files to the correct S3 bucket", func() {
					timestamp := getCurrentTimestamp()
					_, cliCommand := runBackupWithConfig(backupExecutablePath, backupConfigPath)
					for _, instanceID := range instanceIDs {
						Expect(cliCommand).To(MatchRegexp(cliParams(*backupConfig, bucket.Name, timestamp, instanceID, "shared-vm", s3TestServer.URL())))
					}
				})

				It("causes redis to SAVE/BGSAVE before backing up", func() {
					instanceID := instanceIDs[0]
					confPath := filepath.Join(brokerConfig.RedisConfiguration.InstanceDataDirectory, instanceID, "redis.conf")
					lastSaveTime := getLastSaveTime(instanceID, confPath)
					runBackupWithConfig(backupExecutablePath, backupConfigPath)
					Expect(getLastSaveTime(instanceID, confPath)).To(BeNumerically(">", lastSaveTime))
				})

				It("creates the bucket if it does not exist", func() {
					err := bucket.DelBucket()
					Ω(err).NotTo(HaveOccurred())

					buckets, err := client.ListBuckets()
					Expect(err).ToNot(HaveOccurred())
					Expect(buckets.Buckets).To(HaveLen(0))

					backupSession, _ := runBackupWithConfig(backupExecutablePath, backupConfigPath)
					backupSession.Wait(time.Second * 10)

					buckets, err = client.ListBuckets()
					Expect(err).ToNot(HaveOccurred())
					Expect(buckets.Buckets).To(HaveLen(1))
					Expect(buckets.Buckets[0].Name).To(Equal(backupConfig.S3Configuration.BucketName))
				})
			})

			Context("when an instance backup fails", func() {
				It("still backs up the other instances", func() {
					helpers.KillRedisProcess(instanceIDs[0], brokerConfig)

					timestamp := getCurrentTimestamp()
					backupSession, cliCommand := runBackupWithConfig(backupExecutablePath, backupConfigPath)
					backupExitStatusCode := backupSession.ExitCode()
					Ω(backupExitStatusCode).Should(Equal(1))

					Expect(cliCommand).To(MatchRegexp(cliParams(*backupConfig, bucket.Name, timestamp, instanceIDs[1], "shared-vm", s3TestServer.URL())))
				})
			})
		})
	})
})

func cliParams(backupConfig backupconfig.Config, bucketName, timestamp, instanceID, planName, endpointUrl string) string {
	filePath := path.Join(backupConfig.RedisDataDirectory, "[a-z0-9\\-]+")
	if planName == "shared-vm" {
		filePath = path.Join(backupConfig.RedisDataDirectory, instanceID, "db", "[a-z0-9\\-]+")
	}

	args := []string{
		"s3",
		"cp",
		filePath,
		fmt.Sprintf("s3://%s%s", bucketName, backupFilename(backupConfig.S3Configuration.Path, timestamp, instanceID, planName)),
		"--endpoint-url",
		endpointUrl,
	}

	return strings.Join(args, " ")
}

func getCurrentTimestamp() string {
	// http://golang.org/pkg/time/#pkg-constants if you need to understand these crazy layouts
	const desiredTimeLayout = "200601021504"
	const secondsTimeLayout = "05"

	// delay until the next minute if we're in the last 55 seconds of the current one
	seconds, _ := strconv.Atoi(time.Now().Format(secondsTimeLayout))
	for seconds > 55 {
		time.Sleep(time.Second)
		seconds, _ = strconv.Atoi(time.Now().Format(secondsTimeLayout))
	}

	return time.Now().Format(desiredTimeLayout)
}

func backupFilename(path, timestamp, instanceID, planName string) string {
	const datePathLayout = "2006/01/02"
	return fmt.Sprintf(
		"%s/%s/%s_%s_%s_redis_backup",
		path,
		time.Now().Format(datePathLayout),
		timestamp,
		instanceID,
		planName)
}

func getLastSaveTime(instanceID string, configPath string) int64 {
	status, bindingBytes := brokerClient.BindInstance(instanceID, uuid.New())
	Ω(status).To(Equal(http.StatusCreated))
	bindingResponse := map[string]interface{}{}
	json.Unmarshal(bindingBytes, &bindingResponse)
	credentials := bindingResponse["credentials"].(map[string]interface{})

	redisClient, err := client.Connect(
		client.Host(credentials["host"].(string)),
		client.Port(int(credentials["port"].(float64))),
		client.Password(credentials["password"].(string)),
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
	redisClient := helpers.BuildRedisClient(port, credentials["host"].(string), credentials["password"].(string))
	defer redisClient.Close()
	for i := 0; i < 20; i++ {
		_, err := redisClient.Do("SET", fmt.Sprintf("foo%d", i), fmt.Sprintf("bar%d", i))
		Ω(err).ToNot(HaveOccurred())
	}
	_, err := redisClient.Do("SAVE")
	Ω(err).ToNot(HaveOccurred())
}

func writeConfig(config *backupconfig.Config, path string) {
	configFile, err := os.Create(path)
	Ω(err).ToNot(HaveOccurred())
	encoder := candiedyaml.NewEncoder(configFile)
	err = encoder.Encode(config)
	Ω(err).ToNot(HaveOccurred())
}

func swapS3ValuesInBackupConfig(config *backupconfig.Config, path, newEndpointURL, newCliPath string) string {
	oldEndpointURL := config.S3Configuration.EndpointUrl
	config.S3Configuration.EndpointUrl = newEndpointURL
	config.AwsCLIPath = newCliPath
	writeConfig(config, path)
	return oldEndpointURL
}

func runBackupWithConfig(executablePath, configPath string) (*gexec.Session, string) {
	cmd := exec.Command(executablePath)
	cmd.Stdout = GinkgoWriter
	cmd.Stderr = GinkgoWriter
	cliOutputFile, err := ioutil.TempFile(".", "s3_upload_command_args")
	Expect(err).ToNot(HaveOccurred())

	cmd.Env = append(cmd.Env, "BACKUP_CONFIG_PATH="+configPath, "FAKE_CLI_OUTPUT_PATH="+cliOutputFile.Name())
	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).ToNot(HaveOccurred())
	session.Wait(time.Second * 10)

	cliOutputFileContents, err := ioutil.ReadFile(cliOutputFile.Name())
	Expect(err).ToNot(HaveOccurred())
	os.Remove(cliOutputFile.Name())

	return session, string(cliOutputFileContents)
}

func configureS3ClientAndBucket(backupConfig *backupconfig.Config) (*s3.S3, *s3.Bucket) {
	client := s3.New(aws.Auth{
		AccessKey: backupConfig.S3Configuration.AccessKeyId,
		SecretKey: backupConfig.S3Configuration.SecretAccessKey,
	}, aws.Region{
		Name:                 "custom-region",
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
