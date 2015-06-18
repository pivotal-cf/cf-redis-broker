package backup_integration_test

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	goamz "github.com/mitchellh/goamz/s3"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/cf-redis-broker/integration"
	"github.com/pivotal-cf/cf-redis-broker/integration/helpers"
	"github.com/pivotal-cf/cf-redis-broker/s3"
	"github.com/pivotal-golang/lager"
)

var _ = Describe("backups", func() {

	Context("when S3 is not configured", func() {
		It("exits with status code 0", func() {
			configFile := helpers.AssetPath("empty-backup.yml")
			backupExitCode := runBackup(configFile)
			Expect(backupExitCode).Should(Equal(0))
		})
	})

	Context("when S3 is configured correctly", func() {
		var (
			configDir          string
			dataDir            string
			logDir             string
			configFile         string
			redisConfig        string
			planName           string
			awsAccessKey       string
			awsSecretAccessKey string
		)

		BeforeEach(func() {
			configDir, dataDir, logDir = helpers.CreateTestDirs()
		})

		JustBeforeEach(func() {
			awsAccessKey = os.Getenv("AWS_ACCESS_KEY_ID")
			if awsAccessKey == "" {
				panic("AWS_ACCESS_KEY_ID not provided as Env Var")
			}
			awsSecretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
			if awsSecretAccessKey == "" {
				panic("AWS_SECRET_ACCESS_KEY not provided as Env Var")
			}
			templateData := &helpers.TemplateData{
				DataDir:            dataDir,
				ConfigDir:          configDir,
				LogDir:             logDir,
				Host:               "127.0.0.1",
				Port:               integration.RedisPort,
				AwsAccessKey:       awsAccessKey,
				AwsSecretAccessKey: awsSecretAccessKey,
				PlanName:           planName,
				BrokerHost:         brokerHost,
				BrokerPort:         brokerPort,
			}
			redisConfig = filepath.Join(configDir, "redis.conf")
			err := helpers.HandleTemplate(
				helpers.AssetPath("redis.conf.template"),
				redisConfig,
				templateData,
			)
			if err != nil {
				panic(err)
			}
			configFile = filepath.Join(configDir, "working-backup.yml")
			err = helpers.HandleTemplate(
				helpers.AssetPath("working-backup.yml.template"),
				configFile,
				templateData,
			)
			if err != nil {
				panic(err)
			}

			redisRunner = &integration.RedisRunner{}
			redisRunner.Start([]string{redisConfig})
		})

		AfterEach(func() {
			redisRunner.Stop()
			helpers.RemoveTestDirs(configDir, dataDir, logDir)
			cleanupS3(awsAccessKey, awsSecretAccessKey)
		})

		Context("when its a dedicated instance to back up", func() {
			BeforeEach(func() {
				planName = "dedicated-vm"
			})

			It("creates a dump.rdb file in the redis data dir", func() {
				backupExitCode := runBackup(configFile)
				Expect(backupExitCode).Should(Equal(0))
				_, err := os.Stat(filepath.Join(dataDir, "dump.rdb"))
				Expect(err).ToNot(HaveOccurred())
			})

			It("uploads the dump.rdb file to the correct S3 bucket", func() {
				backupExitCode := runBackup(configFile)
				Expect(backupExitCode).To(Equal(0))
				apiClient := newApiClient(awsAccessKey, awsSecretAccessKey)

				m, err := apiClient.Bucket("redis-backup-test").GetBucketContents()
				Expect(err).ToNot(HaveOccurred())
				Expect(len(*m)).To(Equal(1))
				for k, _ := range *m {
					Expect(k).To(ContainSubstring("this_is_an_instance_id_dedicated-vm"))
				}
			})

			// Context("when broker is not responding", func() {
			// 	It("returns non-zero exit code", func() {
			// 	})
			// })

			// Context("when broker returns an error", func() {
			// 	It("returns non-zero exit code", func() {
			// 	})
			// })

			XContext("when the instance backup fails", func() {
				It("returns non-zero exit code", func() {
				})
			})
		})

		XContext("when there are shared-vm instances to back up", func() {
			Context("when the backup command completes successfully", func() {
				It("exits with status code 0", func() {
				})

				It("uploads a dump.rdb file to S3 for each Redis instance", func() {
				})

				It("creates a dump.rdb file for each Redis instance", func() {
				})
			})

			Context("when an instance backup fails", func() {
				It("still backs up the other instances", func() {
				})
			})
		})
	})
})

func newApiClient(awsAccessKey, awsSecretAccessKey string) *goamz.S3 {
	logger := lager.NewLogger("logger")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))
	return s3.NewClient("https://s3.amazonaws.com", awsAccessKey, awsSecretAccessKey, logger).ApiClient()
}

func cleanupS3(awsAccessKey, awsSecretAccessKey string) {
	apiClient := newApiClient(awsAccessKey, awsSecretAccessKey)

	m, err := apiClient.Bucket("redis-backup-test").GetBucketContents()
	panicOnError(err)
	for k, _ := range *m {
		panicOnError(apiClient.Bucket("redis-backup-test").Del(k))
	}
	year := time.Now().Format("2006")
	month := time.Now().Format("01")
	day := time.Now().Format("02")
	panicOnError(apiClient.Bucket("redis-backup-test").Del(fmt.Sprintf("%v/%v/%v", year, month, day)))
	panicOnError(apiClient.Bucket("redis-backup-test").Del(fmt.Sprintf("%v/%v", year, month)))
	panicOnError(apiClient.Bucket("redis-backup-test").Del(fmt.Sprintf("%v", year)))
}

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
