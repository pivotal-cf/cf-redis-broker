package backup_test

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-cf/cf-redis-broker/broker"
	"github.com/pivotal-cf/cf-redis-broker/instance"
	"github.com/pivotal-cf/cf-redis-broker/instance/backup"
	"github.com/pivotal-cf/cf-redis-broker/instance/backup/fakes"
	"github.com/pivotal-cf/cf-redis-broker/redisconf"
	"github.com/pivotal-golang/lager"
	. "github.com/st3v/glager"
)

var _ = Describe("Instance Backuper", func() {
	var (
		providerFactory   *fakes.FakeProviderFactory
		redisClient       *fakes.FakeRedisClient
		redisBackuper     *fakes.FakeRedisBackuper
		redisConfigFinder *fakes.FakeRedisConfigFinder
		instanceIDLocator *fakes.FakeInstanceIDLocator
		backupConfig      *backup.BackupConfig
		logger            lager.Logger
		log               *gbytes.Buffer
	)

	BeforeEach(func() {
		log = gbytes.NewBuffer()
		logger = lager.NewLogger("logger")
		logger.RegisterSink(lager.NewWriterSink(log, lager.DEBUG))

		providerFactory = new(fakes.FakeProviderFactory)

		redisClient = new(fakes.FakeRedisClient)
		providerFactory.RedisClientProviderReturns(redisClient, nil)

		redisBackuper = new(fakes.FakeRedisBackuper)
		providerFactory.RedisBackuperProviderReturns(redisBackuper)

		redisConfigFinder = new(fakes.FakeRedisConfigFinder)
		providerFactory.RedisConfigFinderProviderReturns(redisConfigFinder)

		instanceIDLocator = new(fakes.FakeInstanceIDLocator)
		providerFactory.SharedInstanceIDLocatorProviderReturns(instanceIDLocator)
		providerFactory.DedicatedInstanceIDLocatorProviderReturns(instanceIDLocator)

		providerFactory.TimeProviderStub = func() time.Time {
			now, _ := time.Parse("200601020304", "200102030405")
			return now
		}

		var err error
		backupConfig, err = backup.LoadBackupConfig(filepath.Join("assets", "backup.yml"))
		Expect(err).ToNot(HaveOccurred())

		backupConfig.PlanName = broker.PlanNameShared
	})

	Describe(".Backup", func() {
		var backupResults []backup.BackupResult

		JustBeforeEach(func() {
			backuper, err := backup.NewInstanceBackuper(
				*backupConfig,
				logger,
				backup.InjectTimeProvider(providerFactory.TimeProvider),
				backup.InjectRedisClientProvider(providerFactory.RedisClientProvider),
				backup.InjectRedisBackuperProvider(providerFactory.RedisBackuperProvider),
				backup.InjectRedisConfigFinderProvider(providerFactory.RedisConfigFinderProvider),
				backup.InjectSharedInstanceIDLocatorProvider(providerFactory.SharedInstanceIDLocatorProvider),
				backup.InjectDedicatedInstanceIDLocatorProvider(providerFactory.DedicatedInstanceIDLocatorProvider),
			)

			Expect(err).ToNot(HaveOccurred())

			backupResults = backuper.Backup()
		})

		Context("when there are no redis configs in the config root", func() {
			It("does returns an empty slice of BackupResult", func() {
				Expect(backupResults).ToNot(BeNil())
				Expect(backupResults).To(BeEmpty())
			})
		})

		Context("when there is one redis config in the config root", func() {
			var (
				expectedPath         = "path/to/the/redis/config"
				expectedInstanceID   = "some-instance-id"
				expectedTargetPath   string
				expectedRedisConfigs = []instance.RedisConfig{
					instance.RedisConfig{
						Path: expectedPath,
						Conf: redisconf.New(
							redisconf.Param{"bind", "8.8.8.8"},
							redisconf.Param{"port", "1234"},
						),
					},
				}
			)

			BeforeEach(func() {
				redisConfigFinder.FindReturns(expectedRedisConfigs, nil)
				instanceIDLocator.LocateIDReturns(expectedInstanceID, nil)
				expectedFilename := fmt.Sprintf("200102030405_%s_%s", expectedInstanceID, backupConfig.PlanName)
				expectedTargetPath = filepath.Join(
					backupConfig.S3Config.Path,
					"2001",
					"02",
					"03",
					expectedFilename,
				)
			})

			It("does return a single BackupResult", func() {
				Expect(backupResults).To(HaveLen(1))
			})

			It("does return a BackupResult with no error", func() {
				Expect(backupResults[0].Err).To(BeNil())
			})

			It("does return a BackupResult with the correct instance ID", func() {
				Expect(backupResults[0].InstanceID).To(Equal(expectedInstanceID))
			})

			It("does return a BackupResult with the correct path", func() {
				Expect(backupResults[0].RedisConfigPath).To(Equal(expectedPath))
			})

			It("does return a BackupResult with the correct node ip", func() {
				Expect(backupResults[0].NodeIP).To(Equal(backupConfig.NodeIP))
			})

			It("does connect to redis", func() {
				Expect(providerFactory.RedisClientProviderCallCount()).To(Equal(1))
			})

			It("does perform a redis backup", func() {
				Expect(redisBackuper.BackupCallCount()).To(Equal(1))
			})

			It("does perform a redis backup using the correct redis client", func() {
				actualClient, _ := redisBackuper.BackupArgsForCall(0)
				Expect(actualClient).To(Equal(redisClient))
			})

			It("does perform a redis backup using the correct targetPath", func() {
				_, actualTargetPath := redisBackuper.BackupArgsForCall(0)
				Expect(actualTargetPath).To(Equal(expectedTargetPath))
			})

			It("provides logging", func() {
				expectedLogData := Data(
					"redis_config_path", expectedRedisConfigs[0].Path,
					"node_ip", backupConfig.NodeIP,
				)

				Expect(log).To(ContainSequence(
					Info(
						Action("logger.instance-backup"),
						Data("event", "starting"),
						expectedLogData,
					),
					Info(
						Action("logger.instance-backup.locate-iid"),
						Data("event", "starting"),
						expectedLogData,
					),
					Info(
						Action("logger.instance-backup.locate-iid"),
						Data("event", "done"),
						Data("instance_id", expectedInstanceID),
						expectedLogData,
					),
					Info(
						Action("logger.instance-backup.redis-connect"),
						Data("event", "starting"),
						Data("redis_address", "8.8.8.8:1234"),
						expectedLogData,
					),
					Info(
						Action("logger.instance-backup.redis-connect"),
						Data("event", "done"),
						Data("redis_address", "8.8.8.8:1234"),
						expectedLogData,
					),
					Info(
						Action("logger.instance-backup.redis-backup"),
						Data("event", "starting"),
						Data("target_path", expectedTargetPath),
						expectedLogData,
					),
					Info(
						Action("logger.instance-backup.redis-backup"),
						Data("event", "done"),
						Data("target_path", expectedTargetPath),
						expectedLogData,
					),
					Info(
						Action("logger.instance-backup"),
						Data("event", "done"),
						expectedLogData,
					),
				))
			})

			Context("when locating the instance ID fails", func() {
				var expectedErr = errors.New("eaten by a grue")

				BeforeEach(func() {
					instanceIDLocator.LocateIDReturns("", expectedErr)
				})

				It("reports the error in the BackupResult", func() {
					Expect(backupResults[0].Err).To(Equal(expectedErr))
				})

				It("logs the error", func() {
					expectedLogData := Data(
						"redis_config_path", expectedRedisConfigs[0].Path,
						"node_ip", backupConfig.NodeIP,
					)
					Expect(log).To(ContainSequence(
						Info(
							Action("logger.instance-backup.locate-iid"),
							Data("event", "starting"),
							expectedLogData,
						),
						Error(
							expectedErr,
							Action("logger.instance-backup.locate-iid"),
							Data("event", "failed"),
							expectedLogData,
						),
					))
				})

				It("does not connect to redis", func() {
					Expect(providerFactory.RedisClientProviderCallCount()).To(Equal(0))
				})

				It("does not perform a redis backup", func() {
					Expect(redisBackuper.BackupCallCount()).To(Equal(0))
				})
			})

			Context("when connecting to redis fails", func() {
				var expectedErr = errors.New("lost in time and space")

				BeforeEach(func() {
					providerFactory.RedisClientProviderReturns(nil, expectedErr)
				})

				It("reports the error in the BackupResult", func() {
					Expect(backupResults[0].Err).To(Equal(expectedErr))
				})

				It("logs the error", func() {
					expectedLogData := Data(
						"redis_config_path", expectedRedisConfigs[0].Path,
						"node_ip", backupConfig.NodeIP,
					)
					Expect(log).To(ContainSequence(
						Info(
							Action("logger.instance-backup.redis-connect"),
							Data("event", "starting"),
							expectedLogData,
						),
						Error(
							expectedErr,
							Action("logger.instance-backup.redis-connect"),
							Data("event", "failed"),
							expectedLogData,
						),
					))
				})

				It("does not perform a redis backup", func() {
					Expect(redisBackuper.BackupCallCount()).To(Equal(0))
				})
			})

			Context("when the redis backup fails", func() {
				var expectedErr = errors.New("Communist state expected")

				BeforeEach(func() {
					redisBackuper.BackupReturns(expectedErr)
				})

				It("reports the error in the BackupResult", func() {
					Expect(backupResults[0].Err).To(Equal(expectedErr))
				})

				It("logs the error", func() {
					expectedLogData := Data(
						"redis_config_path", expectedRedisConfigs[0].Path,
						"node_ip", backupConfig.NodeIP,
					)
					Expect(log).To(ContainSequence(
						Info(
							Action("logger.instance-backup.redis-backup"),
							Data("event", "starting"),
							expectedLogData,
						),
						Error(
							expectedErr,
							Action("logger.instance-backup.redis-backup"),
							Data("event", "failed"),
							expectedLogData,
						),
					))
				})
			})
		})
	})

	Describe("NewInstanceBackuper", func() {
		var (
			backuper    backup.InstanceBackuper
			backuperErr error
		)

		JustBeforeEach(func() {
			backuper, backuperErr = backup.NewInstanceBackuper(
				*backupConfig,
				logger,
				backup.InjectTimeProvider(providerFactory.TimeProvider),
				backup.InjectRedisClientProvider(providerFactory.RedisClientProvider),
				backup.InjectRedisBackuperProvider(providerFactory.RedisBackuperProvider),
				backup.InjectRedisConfigFinderProvider(providerFactory.RedisConfigFinderProvider),
				backup.InjectSharedInstanceIDLocatorProvider(providerFactory.SharedInstanceIDLocatorProvider),
				backup.InjectDedicatedInstanceIDLocatorProvider(providerFactory.DedicatedInstanceIDLocatorProvider),
			)
		})

		It("creates the redis backuper with the right timeout", func() {
			expectedTimeout := time.Duration(backupConfig.SnapshotTimeoutSeconds) * time.Second
			actualTimeout, _, _, _, _, _, _ := providerFactory.RedisBackuperProviderArgsForCall(0)
			Expect(actualTimeout).To(Equal(expectedTimeout))
		})

		It("creates the redis backuper with the right bucket name", func() {
			_, actualBucketName, _, _, _, _, _ := providerFactory.RedisBackuperProviderArgsForCall(0)
			Expect(actualBucketName).To(Equal(backupConfig.S3Config.BucketName))
		})

		It("creates the redis backuper with the right endpoint", func() {
			_, _, actualEndpoint, _, _, _, _ := providerFactory.RedisBackuperProviderArgsForCall(0)
			Expect(actualEndpoint).To(Equal(backupConfig.S3Config.EndpointUrl))
		})

		It("creates the redis backuper with the right access key", func() {
			_, _, _, actualAccessKey, _, _, _ := providerFactory.RedisBackuperProviderArgsForCall(0)
			Expect(actualAccessKey).To(Equal(backupConfig.S3Config.AccessKeyId))
		})

		It("creates the redis backuper with the right secret key", func() {
			_, _, _, _, actualSecretKey, _, _ := providerFactory.RedisBackuperProviderArgsForCall(0)
			Expect(actualSecretKey).To(Equal(backupConfig.S3Config.SecretAccessKey))
		})

		It("creates the redis backuper with the right logger", func() {
			_, _, _, _, _, actualLogger, _ := providerFactory.RedisBackuperProviderArgsForCall(0)
			Expect(actualLogger).To(Equal(logger))
		})

		It("does not inject anything into the redis backuper", func() {
			_, _, _, _, _, _, injectors := providerFactory.RedisBackuperProviderArgsForCall(0)
			Expect(injectors).To(BeEmpty())
		})

		It("creates the redis config finder with the right root path", func() {
			rootPath, _ := providerFactory.RedisConfigFinderProviderArgsForCall(0)
			Expect(rootPath).To(Equal(backupConfig.RedisConfigRoot))
		})

		It("creates the redis config finder with the right filename", func() {
			_, filename := providerFactory.RedisConfigFinderProviderArgsForCall(0)
			Expect(filename).To(Equal(backupConfig.RedisConfigFilename))
		})

		It("calls the redis config finder", func() {
			Expect(redisConfigFinder.FindCallCount()).To(Equal(1))
		})

		It("provides logging", func() {
			Expect(log).To(ContainSequence(
				Info(
					Action("logger.init-instance-backuper"),
					Data("event", "starting"),
				),
				Info(
					Action("logger.init-instance-backuper"),
					Data("event", "done"),
				),
			))
		})

		Context("when the redis config finder fails", func() {
			var expectedErr = errors.New("some-error")

			BeforeEach(func() {
				redisConfigFinder.FindReturns(nil, expectedErr)
			})

			It("returns the error", func() {
				Expect(backuperErr).To(Equal(expectedErr))
			})

			It("logs the error", func() {
				Expect(log).To(ContainSequence(
					Info(
						Action("logger.init-instance-backuper"),
						Data("event", "starting"),
					),
					Error(
						expectedErr,
						Action("logger.redis-config-finder"),
						Data("event", "failed"),
						Data("root_path", backupConfig.RedisConfigRoot),
						Data("file_name", backupConfig.RedisConfigFilename),
					),
					Error(
						expectedErr,
						Action("logger.init-instance-backuper"),
						Data("event", "failed"),
					),
				))
			})
		})

		Context("when the config specifies the shared plan name", func() {
			BeforeEach(func() {
				backupConfig.PlanName = broker.PlanNameShared
			})

			It("does not return an error", func() {
				Expect(backuperErr).ToNot(HaveOccurred())
			})

			It("uses the correct instance id locator", func() {
				Expect(providerFactory.SharedInstanceIDLocatorProviderCallCount()).To(Equal(1))
				Expect(providerFactory.DedicatedInstanceIDLocatorProviderCallCount()).To(Equal(0))
			})
		})

		Context("when the config specifies the dedicated plan name", func() {
			BeforeEach(func() {
				backupConfig.PlanName = broker.PlanNameDedicated
			})

			It("does not return an error", func() {
				Expect(backuperErr).ToNot(HaveOccurred())
			})

			It("uses the correct instance id locator", func() {
				Expect(providerFactory.SharedInstanceIDLocatorProviderCallCount()).To(Equal(0))
				Expect(providerFactory.DedicatedInstanceIDLocatorProviderCallCount()).To(Equal(1))
			})
		})

		Context("when the config specifies an invalid plan name", func() {
			BeforeEach(func() {
				backupConfig.PlanName = "foobar"
			})

			It("returns an error", func() {
				Expect(backuperErr).To(HaveOccurred())
				Expect(backuperErr.Error()).To(ContainSubstring("Unknown plan"))
			})

			It("logs the error", func() {
				Expect(log).To(ContainSequence(
					Info(
						Action("logger.init-instance-backuper"),
						Data("event", "starting"),
					),
					Error(
						backuperErr,
						Action("logger.init-iid-locator"),
						Data("event", "failed"),
						Data("plan_name", backupConfig.PlanName),
					),
					Error(
						backuperErr,
						Action("logger.init-instance-backuper"),
						Data("event", "failed"),
					),
				))
			})
		})
	})
})
