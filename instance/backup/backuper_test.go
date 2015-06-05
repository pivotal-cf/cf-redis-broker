package backup_test

import (
	"errors"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-cf/cf-redis-broker/broker"
	"github.com/pivotal-cf/cf-redis-broker/instance/backup"
	"github.com/pivotal-cf/cf-redis-broker/instance/backup/fakes"
	"github.com/pivotal-golang/lager"
	. "github.com/st3v/glager"
)

var _ = Describe("Instance Backuper", func() {
	Describe("NewInstanceBackuper", func() {
		var (
			providerFactory            *fakes.FakeProviderFactory
			redisBackuper              *fakes.FakeRedisBackuper
			redisConfigFinder          *fakes.FakeRedisConfigFinder
			sharedInstanceIDLocator    *fakes.FakeInstanceIDLocator
			dedicatedInstanceIDLocator *fakes.FakeInstanceIDLocator
			backupConfig               *backup.BackupConfig
			logger                     lager.Logger
			log                        *gbytes.Buffer
			backuper                   backup.InstanceBackuper
			backuperErr                error
		)

		BeforeEach(func() {
			log = gbytes.NewBuffer()
			logger = lager.NewLogger("logger")
			logger.RegisterSink(lager.NewWriterSink(log, lager.DEBUG))

			providerFactory = new(fakes.FakeProviderFactory)

			redisBackuper = new(fakes.FakeRedisBackuper)
			providerFactory.RedisBackuperProviderReturns(redisBackuper)

			redisConfigFinder = new(fakes.FakeRedisConfigFinder)
			providerFactory.RedisConfigFinderProviderReturns(redisConfigFinder)

			sharedInstanceIDLocator = new(fakes.FakeInstanceIDLocator)
			providerFactory.SharedInstanceIDLocatorProviderReturns(sharedInstanceIDLocator)

			dedicatedInstanceIDLocator = new(fakes.FakeInstanceIDLocator)
			providerFactory.DedicatedInstanceIDLocatorProviderReturns(dedicatedInstanceIDLocator)

			providerFactory.TimeProviderStub = func() time.Time {
				now, _ := time.Parse("200601020304", "200102030405")
				return now
			}

			var err error
			backupConfig, err = backup.LoadBackupConfig(filepath.Join("assets", "backup.yml"))
			Expect(err).ToNot(HaveOccurred())

			backupConfig.PlanName = broker.PlanNameShared
		})

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
