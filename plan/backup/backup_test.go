package backup

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-cf/cf-redis-broker/plan"
	redis "github.com/pivotal-cf/cf-redis-broker/redis/client"
	redisFakes "github.com/pivotal-cf/cf-redis-broker/redis/client/fakes"
	"github.com/pivotal-cf/cf-redis-broker/redisconf"
	"github.com/pivotal-golang/lager"
	. "github.com/st3v/glager"
)

type iidProvider struct {
	ExpectedInstanceIDs      []string
	ExpectedInstanceIDErrors []error
	ActualInstanceIDArgs     [][]string
}

func (p *iidProvider) InstanceID(redisConfigPath, nodeIP string) (string, error) {
	if p.ActualInstanceIDArgs == nil {
		p.ActualInstanceIDArgs = [][]string{}
	}

	p.ActualInstanceIDArgs = append(p.ActualInstanceIDArgs, []string{redisConfigPath, nodeIP})
	idx := len(p.ActualInstanceIDArgs) - 1
	return p.ExpectedInstanceIDs[idx], p.ExpectedInstanceIDErrors[idx]
}

var _ = Describe("backup", func() {
	FDescribe(".Backup", func() {
		var (
			log              *gbytes.Buffer
			backupConfigPath string
			backupResult     []Result
			backupErr        error

			backedUpClients []redis.Client
			snapshotTimeout time.Duration
			s3BucketNames   []string
			s3TargetPaths   []string
			// s3Endpoints     []string
			// s3AccessKeys    []string
			// s3SecretKeys    []string

			backupConfig         *Config
			redisConfigRoot      string
			redisConfigFilename  string
			redisClient          redis.Client
			redisClientOptions   [][]redis.Option
			redisConfigs         map[string]redisconf.Conf
			redisConfigLoaderErr error
			redisConnectErrors   []error

			expectedInstanceIDs      = []string{"instance-1", "instance-2"}
			expectedInstanceIDErrors = make([]error, 2)

			logger lager.Logger

			instanceIDProvider           *iidProvider
			instanceIDProviderFactoryErr error
		)

		BeforeEach(func() {
			redisConfigLoaderErr = nil
			redisConfigs = map[string]redisconf.Conf{}

			backupConfigPath = filepath.Join("assets", "backup.yml")

			log = gbytes.NewBuffer()
			logger = lager.NewLogger("logger")
			logger.RegisterSink(lager.NewWriterSink(log, lager.DEBUG))

			defaultRedisConfigLoader = func(configRoot, configFilename string) (map[string]redisconf.Conf, error) {
				redisConfigRoot = configRoot
				redisConfigFilename = configFilename
				return redisConfigs, redisConfigLoaderErr
			}

			instanceIDProvider = &iidProvider{
				ExpectedInstanceIDs:      expectedInstanceIDs,
				ExpectedInstanceIDErrors: expectedInstanceIDErrors,
			}

			instanceIDProviderFactoryErr = nil

			defaultInstanceIDProviderFactory = func(config *Config, logger lager.Logger) (plan.IDProvider, error) {
				backupConfig = config
				return instanceIDProvider, instanceIDProviderFactoryErr
			}

			redisClient = &redisFakes.Client{}
			redisClientOptions = [][]redis.Option{}
			redisConnectErrors = []error{}
			defaultRedisClientProvider = func(options ...redis.Option) (redis.Client, error) {
				redisClientOptions = append(redisClientOptions, options)

				var err error
				if len(redisConnectErrors) < len(redisClientOptions) {
					err = nil
				} else {
					err = redisConnectErrors[len(redisClientOptions)-1]
				}

				return redisClient, err
			}

			backedUpClients = []redis.Client{}
			s3TargetPaths = []string{}
			s3BucketNames = []string{}

			defaultRedisBackupFunc = func(
				client redis.Client,
				timeout time.Duration,
				bucketName string,
				targetPath string,
				endpoint string,
				eccessKey string,
				secretKey string,
				logger lager.Logger,
			) error {
				snapshotTimeout = timeout
				s3BucketNames = append(s3BucketNames, bucketName)
				s3TargetPaths = append(s3TargetPaths, targetPath)
				backedUpClients = append(backedUpClients, client)
				return nil
			}
		})

		JustBeforeEach(func() {
			backupResult, backupErr = Backup(backupConfigPath, logger)
		})

		It("provides logging", func() {
			Expect(log).To(ContainSequence(
				Info(
					Action("logger.plan-backup"),
					Data("event", "starting", "backup_config_path", backupConfigPath),
				),
				Info(
					Action("logger.plan-backup"),
					Data("event", "done", "backup_config_path", backupConfigPath),
				),
			))
		})

		It("loads the redis config from the correct root", func() {
			Expect(redisConfigRoot).To(Equal("/the/path/to/redis/config"))
		})

		It("loads the redis config using the correct filename", func() {
			Expect(redisConfigFilename).To(Equal("redis-config-filename"))
		})

		It("creates the InstanceID provider using the correct backup config", func() {
			Expect(backupConfig.PlanName).To(Equal("plan-name"))
		})

		Context("when loading the backup config fails", func() {
			BeforeEach(func() {
				backupConfigPath = "/path/to/nowhere"
			})

			It("returns the error", func() {
				Expect(backupErr).To(HaveOccurred())
				Expect(backupErr).To(BeAssignableToTypeOf(&os.PathError{}))
			})

			It("logs the error", func() {
				Expect(log).To(ContainSequence(
					Info(
						Action("logger.plan-backup"),
						Data("event", "starting", "backup_config_path", backupConfigPath),
					),
					Error(
						nil,
						Action("logger.plan-backup"),
						Data("event", "failed", "backup_config_path", backupConfigPath),
					),
				))
			})
		})

		Context("when an empty config path is being passed", func() {
			BeforeEach(func() {
				backupConfigPath = ""
			})

			It("returns an error", func() {
				Expect(backupErr).To(HaveOccurred())
			})

			It("logs the error", func() {
				Expect(log).To(ContainSequence(
					Info(
						Action("logger.plan-backup"),
						Data("event", "starting", "backup_config_path", backupConfigPath),
					),
					Error(
						nil,
						Action("logger.plan-backup"),
						Data("event", "failed", "backup_config_path", backupConfigPath),
					),
				))
			})
		})

		Context("when the redis configs are not successfully loaded", func() {
			BeforeEach(func() {
				redisConfigLoaderErr = errors.New("Error loading config")
			})

			It("returns an error", func() {
				Expect(backupErr).To(HaveOccurred())
				Expect(backupErr).To(Equal(redisConfigLoaderErr))
			})

			It("logs the error", func() {
				Expect(log).To(ContainSequence(
					Info(
						Action("logger.plan-backup"),
						Data("event", "starting", "backup_config_path", backupConfigPath),
					),
					Error(
						redisConfigLoaderErr,
						Action("logger.plan-backup.load-redis-configs"),
						Data("event", "failed", "redis_config_root", "/the/path/to/redis/config", "redis_config_filename", "redis-config-filename"),
					),
				))
			})
		})

		Context("when getting the instance ID provider fails", func() {
			BeforeEach(func() {
				instanceIDProviderFactoryErr = errors.New("some-error")
			})

			It("returns an error", func() {
				Expect(backupErr).To(HaveOccurred())
				Expect(backupErr).To(Equal(instanceIDProviderFactoryErr))
			})

			It("logs the error", func() {
				Expect(log).To(ContainSequence(
					Info(
						Action("logger.plan-backup"),
						Data("event", "starting", "backup_config_path", backupConfigPath),
					),
					Error(
						instanceIDProviderFactoryErr,
						Action("logger.plan-backup.get-instance-id-provider"),
						Data("event", "failed", "plan_name", "plan-name"),
					),
				))
			})
		})

		Context("for the shared plan", func() {
			Context("when there are no provisioned redis instances", func() {
				It("does not return an error", func() {
					Expect(backupErr).ToNot(HaveOccurred())
				})

				It("does not call the instance ID provider", func() {
					Expect(instanceIDProvider.ActualInstanceIDArgs).To(BeEmpty())
				})

				It("returns an empty result", func() {
					Expect(backupResult).To(BeEmpty())
				})

				It("does not connect to redis", func() {
					Expect(redisClientOptions).To(BeEmpty())
				})

				It("does not perform a redis backup", func() {
					Expect(backedUpClients).To(BeEmpty())
				})
			})

			Context("when there are two provisioned redis instances", func() {
				var (
					redisClients []redis.Client
				)

				BeforeEach(func() {
					redisClients = []redis.Client{
						&redisFakes.Client{
							Host: "127.0.0.1",
						},
						&redisFakes.Client{
							Host: "127.0.0.2",
						},
					}

					redisConnectErrors = make([]error, 2)

					defaultRedisClientProvider = func(options ...redis.Option) (redis.Client, error) {
						redisClientOptions = append(redisClientOptions, options)
						client := redisClients[len(redisClientOptions)-1]
						err := redisConnectErrors[len(redisClientOptions)-1]
						return client, err
					}

					redisConfigs = map[string]redisconf.Conf{
						"conf1": redisconf.New(redisconf.Param{"bind", "127.0.0.1"}),
						"conf2": redisconf.New(redisconf.Param{"bind", "127.0.0.2"}),
					}

					clock = func() time.Time {
						now, _ := time.Parse("200601020304", "200102030405")
						return now
					}
				})

				It("does not return an error", func() {
					Expect(backupErr).ToNot(HaveOccurred())
				})

				It("does call the instance ID provider for both instances", func() {
					Expect(instanceIDProvider.ActualInstanceIDArgs).To(HaveLen(2))
				})

				It("calls the instance ID provider with the right args", func() {
					Expect(instanceIDProvider.ActualInstanceIDArgs).To(ContainElement([]string{
						"conf1",
						"1.2.3.4",
					}))
					Expect(instanceIDProvider.ActualInstanceIDArgs).To(ContainElement([]string{
						"conf2",
						"1.2.3.4",
					}))
				})

				It("returns two results", func() {
					Expect(backupResult).To(HaveLen(2))
				})

				It("does connect to both redis instances", func() {
					Expect(redisClientOptions).To(HaveLen(2))
				})

				It("uses the correct number of options when connecting to redis", func() {
					Expect(redisClientOptions[0]).To(HaveLen(4))
					Expect(redisClientOptions[1]).To(HaveLen(4))
				})

				It("performs a backup for both redis instances", func() {
					Expect(backedUpClients).To(ContainElement(redisClients[0]))
					Expect(backedUpClients).To(ContainElement(redisClients[1]))
				})

				It("uses the right s3 target path for each instance", func() {
					for _, iid := range instanceIDProvider.ExpectedInstanceIDs {
						expectedTargetPath := fmt.Sprintf(
							"some-s3-path/2001/02/03/200102030405_%s_%s",
							iid,
							"plan-name",
						)
						Expect(s3TargetPaths).To(ContainElement(expectedTargetPath))
					}
				})

				It("uses the right s3 bucket for each instance", func() {
					for i, _ := range instanceIDProvider.ExpectedInstanceIDs {
						Expect(s3BucketNames[i]).To(Equal("some-bucket-name"))
					}
				})

				It("does use the correct snapshot timeout", func() {
					Expect(snapshotTimeout).To(Equal(10 * time.Second))
				})

				Context("when connecting to redis fails for the first instance", func() {
					BeforeEach(func() {
						redisConnectErrors = []error{errors.New("some-error"), nil}
					})

					It("does not return an error", func() {
						Expect(backupErr).ToNot(HaveOccurred())
					})

					It("returns two results", func() {
						Expect(backupResult).To(HaveLen(2))
					})

					It("reports the error for the first instance", func() {
						Expect(backupResult[0].Err).To(Equal(redisConnectErrors[0]))
					})

					It("logs the error for the first instance", func() {
						Expect(log).To(ContainSequence(
							Error(
								redisConnectErrors[0],
								Data("event", "failed"),
								// Data("address", fmt.Sprintf("127.0.0.1:%d", redisconf.DefaultPort)),
							),
						))
					})

					It("does not report an error for the second instance", func() {
						Expect(backupResult[1].Err).To(BeNil())
					})

					It("does not perfom a backup for the first instance", func() {
						Expect(backedUpClients).ToNot(ContainElement(redisClients[0]))
					})

					It("does perform a backup for the second instance", func() {
						Expect(backedUpClients).To(ContainElement(redisClients[1]))
					})
				})
			})
		})
	})

	Describe("old stuff", func() {
		Context("anything else", func() {

			It("loads redis configs from the correct root", func() {

			})

			It("loads redis configs with the correct filename", func() {

			})

			It("uses the correct ID provider", func() {

			})

			It("gets the instance ID for each redis instance", func() {

			})

			It("connects to each redis instance", func() {

			})

			It("closes the connection for each redis instance", func() {

			})

			It("initiates a backup for each instance", func() {

			})

			It("provides logging", func() {

			})

			Context("when parsing the snapshot timeout duration fails", func() {
				It("returns an error", func() {

				})

				It("logs the error", func() {

				})

				It("exists early", func() {

				})
			})

			Context("when loading the redis configs fails", func() {
				It("returns an error", func() {

				})

				It("logs the error", func() {

				})

				It("exists early", func() {

				})
			})

			Context("when connecting to redis fails for one instance", func() {
				It("keeps processing the remaining instances", func() {

				})

				It("returns the error", func() {

				})

				It("logs the error", func() {

				})
			})

			Context("when getting the ID for one instance fails", func() {
				It("keeps processing the remaining instances", func() {

				})

				It("returns the error", func() {

				})

				It("logs the error", func() {

				})
			})

			Context("when backing up one redis instance fails", func() {
				It("keeps processing the remaining instances", func() {

				})

				It("returns the error", func() {

				})

				It("logs the error", func() {

				})
			})
		})

		Context("for the dedicated plan", func() {
			It("loads redis configs from the correct root", func() {

			})

			It("loads redis configs with the correct filename", func() {

			})

			It("uses the correct ID provider", func() {

			})

			It("gets the instance ID for each redis instance", func() {

			})

			It("initiates a backup for each instance", func() {

			})

			It("provides logging", func() {

			})

			Context("when parsing the snapshot timeout duration fails", func() {
				It("returns an error", func() {

				})

				It("logs the error", func() {

				})
			})

			Context("when loading the redis configs fails", func() {
				It("returns an error", func() {

				})

				It("logs the error", func() {

				})
			})

			Context("when connecting to redis fails for one instance", func() {
				It("returns the error", func() {

				})

				It("logs the error", func() {

				})
			})

			Context("when getting the ID for one instance fails", func() {
				It("returns the error", func() {

				})

				It("logs the error", func() {

				})
			})

			Context("when backing up one redis instance fails", func() {
				It("returns the error", func() {

				})

				It("logs the error", func() {

				})
			})
		})

		Context("when the backup config is empty", func() {
			It("returns an error", func() {

			})

			It("logs an error", func() {

			})

			It("exists early", func() {

			})
		})

		Context("when the backup config cannot be loaded", func() {
			It("returns an error", func() {

			})

			It("logs an error", func() {

			})

			It("exists early", func() {

			})
		})
	})
})
