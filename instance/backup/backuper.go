package backup

import (
	"fmt"
	"time"

	"github.com/pivotal-cf/cf-redis-broker/broker"
	"github.com/pivotal-cf/cf-redis-broker/instance"
	"github.com/pivotal-cf/cf-redis-broker/instance/id"
	redisbackup "github.com/pivotal-cf/cf-redis-broker/redis/backup"
	redis "github.com/pivotal-cf/cf-redis-broker/redis/client"
	"github.com/pivotal-golang/lager"
)

type ProviderFactory interface {
	TimeProvider() time.Time
	RedisClientProvider(options ...redis.Option) (redis.Client, error)
	RedisConfigFinderProvider(string, string) instance.RedisConfigFinder
	SharedInstanceIDLocatorProvider(lager.Logger) id.InstanceIDLocator
	DedicatedInstanceIDLocatorProvider(string, string, string, lager.Logger) id.InstanceIDLocator
	RedisBackuperProvider(
		time.Duration, string, string, string, string,
		lager.Logger, ...redisbackup.BackupInjector,
	) redisbackup.RedisBackuper
}

type RedisBackuperProvider func(
	time.Duration, string, string, string, string,
	lager.Logger, ...redisbackup.BackupInjector,
) redisbackup.RedisBackuper

type TimeProvider func() time.Time

type RedisClientProvider func(options ...redis.Option) (redis.Client, error)

type RedisConfigFinderProvider func(string, string) instance.RedisConfigFinder

type SharedInstanceIDLocatorProvider func(lager.Logger) id.InstanceIDLocator

type DedicatedInstanceIDLocatorProvider func(string, string, string, lager.Logger) id.InstanceIDLocator

type BackupResult struct {
	InstanceID      string
	RedisConfigPath string
	NodeIP          string
	Err             error
}

type InstanceBackuper interface {
	Backup() []BackupResult
}

type instanceBackuper struct {
	redisBackuperProvider              RedisBackuperProvider
	redisConfigFinderProvider          RedisConfigFinderProvider
	redisClientProvider                RedisClientProvider
	timeProvider                       TimeProvider
	sharedInstanceIDLocatorProvider    SharedInstanceIDLocatorProvider
	dedicatedInstanceIDLocatorProvider DedicatedInstanceIDLocatorProvider

	redisBackuper     redisbackup.RedisBackuper
	redisConfigs      []instance.RedisConfig
	instanceIDLocator id.InstanceIDLocator
	logger            lager.Logger
}

func NewInstanceBackuper(
	backupConfig BackupConfig,
	logger lager.Logger,
	injectors ...BackupInjector,
) (*instanceBackuper, error) {
	backuper := &instanceBackuper{
		redisConfigFinderProvider:          instance.NewRedisConfigFinder,
		redisBackuperProvider:              redisbackup.NewRedisBackuper,
		redisClientProvider:                redis.Connect,
		sharedInstanceIDLocatorProvider:    id.SharedInstanceIDLocator,
		dedicatedInstanceIDLocatorProvider: id.DedicatedInstanceIDLocator,
		timeProvider:                       time.Now,
		logger:                             logger,
	}

	for _, injector := range injectors {
		injector(backuper)
	}

	logger.Info("init-instance-backuper", lager.Data{"event": "starting"})

	if err := backuper.init(backupConfig); err != nil {
		logger.Error("init-instance-backuper", err, lager.Data{"event": "failed"})
		return nil, err
	}

	logger.Info("init-instance-backuper", lager.Data{"event": "done"})

	return backuper, nil
}

func (b *instanceBackuper) Backup() []BackupResult {
	return nil
}

type BackupInjector func(*instanceBackuper)

func InjectRedisConfigFinderProvider(p RedisConfigFinderProvider) BackupInjector {
	return func(b *instanceBackuper) {
		b.redisConfigFinderProvider = p
	}
}

func InjectRedisClientProvider(p RedisClientProvider) BackupInjector {
	return func(b *instanceBackuper) {
		b.redisClientProvider = p
	}
}

func InjectRedisBackuperProvider(p RedisBackuperProvider) BackupInjector {
	return func(b *instanceBackuper) {
		b.redisBackuperProvider = p
	}
}

func InjectSharedInstanceIDLocatorProvider(p SharedInstanceIDLocatorProvider) BackupInjector {
	return func(b *instanceBackuper) {
		b.sharedInstanceIDLocatorProvider = p
	}
}

func InjectDedicatedInstanceIDLocatorProvider(p DedicatedInstanceIDLocatorProvider) BackupInjector {
	return func(b *instanceBackuper) {
		b.dedicatedInstanceIDLocatorProvider = p
	}
}

func InjectTimeProvider(p TimeProvider) BackupInjector {
	return func(b *instanceBackuper) {
		b.timeProvider = p
	}
}

func (b *instanceBackuper) init(config BackupConfig) error {
	b.redisBackuper = b.redisBackuperProvider(
		time.Duration(config.SnapshotTimeoutSeconds)*time.Second,
		config.S3Config.BucketName,
		config.S3Config.EndpointUrl,
		config.S3Config.AccessKeyId,
		config.S3Config.SecretAccessKey,
		b.logger,
	)

	err := b.initInstanceIDLocator(config)
	if err != nil {
		b.logger.Error("init-iid-locator", err, lager.Data{
			"event":     "failed",
			"plan_name": config.PlanName,
		})
		return err
	}

	redisConfigFinder := b.redisConfigFinderProvider(
		config.RedisConfigRoot,
		config.RedisConfigFilename,
	)

	b.redisConfigs, err = redisConfigFinder.Find()
	if err != nil {
		b.logger.Error("redis-config-finder", err, lager.Data{
			"event":     "failed",
			"root_path": config.RedisConfigRoot,
			"file_name": config.RedisConfigFilename,
		})
		return err
	}

	return nil
}

func (b *instanceBackuper) initInstanceIDLocator(config BackupConfig) error {
	switch config.PlanName {
	case broker.PlanNameShared:
		b.instanceIDLocator = b.sharedInstanceIDLocatorProvider(b.logger)
		return nil
	case broker.PlanNameDedicated:
		b.instanceIDLocator = b.dedicatedInstanceIDLocatorProvider(
			config.BrokerAddress,
			config.BrokerCredentials.Username,
			config.BrokerCredentials.Password,
			b.logger,
		)
		return nil
	}
	return fmt.Errorf("Unknown plan name %s", config.PlanName)
}
