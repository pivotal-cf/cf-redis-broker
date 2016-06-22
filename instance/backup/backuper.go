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

//go:generate counterfeiter -o fakes/provider_factory.go . ProviderFactory
type ProviderFactory interface {
	TimeProvider() time.Time
	RedisClientProvider(options ...redis.Option) (redis.Client, error)
	RedisConfigFinderProvider(string, string) instance.RedisConfigFinder
	SharedInstanceIDLocatorProvider(lager.Logger) id.InstanceIDLocator
	DedicatedInstanceIDLocatorProvider(string, string, string, lager.Logger) id.InstanceIDLocator
	RedisBackuperProvider(
		time.Duration, string, string, string, string, string,
		lager.Logger, ...redisbackup.BackupInjector,
	) redisbackup.RedisBackuper
}

type RedisBackuperProvider func(
	time.Duration, string, string, string, string, string,
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
	backupConfig      BackupConfig
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
		backupConfig:                       backupConfig,
	}

	for _, injector := range injectors {
		injector(backuper)
	}

	logger.Info("init-instance-backuper", lager.Data{"event": "starting"})

	if err := backuper.init(); err != nil {
		logger.Error("init-instance-backuper", err, lager.Data{"event": "failed"})
		return nil, err
	}

	logger.Info("init-instance-backuper", lager.Data{"event": "done"})

	return backuper, nil
}

func (b *instanceBackuper) Backup() []BackupResult {
	backupResults := make([]BackupResult, len(b.redisConfigs))
	for i, redisConfig := range b.redisConfigs {
		backupResults[i] = b.instanceBackup(redisConfig)
	}
	return backupResults
}

func (b *instanceBackuper) instanceBackup(redisConfig instance.RedisConfig) BackupResult {
	result := BackupResult{
		RedisConfigPath: redisConfig.Path,
		NodeIP:          b.backupConfig.NodeIP,
	}

	logger := b.logger.WithData(
		lager.Data{
			"node_ip":           b.backupConfig.NodeIP,
			"redis_config_path": redisConfig.Path,
		},
	)

	logger.Info("instance-backup", lager.Data{"event": "starting"})

	logger.Info("instance-backup.locate-iid", lager.Data{"event": "starting"})
	instanceID, err := b.instanceIDLocator.LocateID(
		redisConfig.Path,
		b.backupConfig.NodeIP,
	)
	if err != nil {
		logger.Error("instance-backup.locate-iid", err, lager.Data{"event": "failed"})
		result.Err = err
		return result
	}

	result.InstanceID = instanceID

	logger.Info("instance-backup.locate-iid", lager.Data{
		"event":       "done",
		"instance_id": instanceID,
	})

	redisAddress := fmt.Sprintf("%s:%d", redisConfig.Conf.Host(), redisConfig.Conf.Port())
	logger.Info("instance-backup.redis-connect", lager.Data{
		"event":         "starting",
		"redis_address": redisAddress,
	})
	redisClient, err := b.redisClientProvider(
		redis.Host(redisConfig.Conf.Host()),
		redis.Port(redisConfig.Conf.Port()),
		redis.Password(redisConfig.Conf.Password()),
		redis.CmdAliases(redisConfig.Conf.CommandAliases()),
	)
	if err != nil {
		logger.Error("instance-backup.redis-connect", err, lager.Data{"event": "failed"})
		result.Err = err
		return result
	}
	defer func() {
		if err := redisClient.Disconnect(); err != nil {
			logger.Error("instance-backup.redis-disconnect", err, lager.Data{"event": "failed"})
		}
	}()
	logger.Info("instance-backup.redis-connect", lager.Data{
		"event":         "done",
		"redis_address": redisAddress,
	})

	targetPath := b.buildTargetPath(instanceID)
	logger.Info("instance-backup.redis-backup", lager.Data{
		"event":       "starting",
		"target_path": targetPath,
	})
	err = b.redisBackuper.Backup(redisClient, targetPath)
	if err != nil {
		logger.Error("instance-backup.redis-backup", err, lager.Data{"event": "failed"})
		result.Err = err
		return result
	}
	logger.Info("instance-backup.redis-backup", lager.Data{
		"event":       "done",
		"target_path": targetPath,
	})

	logger.Info("instance-backup", lager.Data{"event": "done"})
	return result
}

func (b *instanceBackuper) buildTargetPath(instanceID string) string {
	now := b.timeProvider()
	year := now.Year()
	month := int(now.Month())
	day := now.Day()

	filename := fmt.Sprintf(
		"%s_%s_%s",
		now.Format("200601021504"),
		instanceID,
		b.backupConfig.PlanName,
	)

	return fmt.Sprintf("%s/%d/%02d/%02d/%s",
		b.backupConfig.S3Config.Path,
		year,
		month,
		day,
		filename,
	)
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

func (b *instanceBackuper) init() error {
	b.redisBackuper = b.redisBackuperProvider(
		time.Duration(b.backupConfig.SnapshotTimeoutSeconds)*time.Second,
		b.backupConfig.S3Config.BucketName,
		b.backupConfig.S3Config.EndpointUrl,
		b.backupConfig.S3Config.AccessKeyId,
		b.backupConfig.S3Config.SecretAccessKey,
		b.backupConfig.BackupTmpDir,
		b.logger,
	)

	err := b.initInstanceIDLocator()
	if err != nil {
		b.logger.Error("init-iid-locator", err, lager.Data{
			"event":     "failed",
			"plan_name": b.backupConfig.PlanName,
		})
		return err
	}

	redisConfigFinder := b.redisConfigFinderProvider(
		b.backupConfig.RedisConfigRoot,
		b.backupConfig.RedisConfigFilename,
	)

	b.redisConfigs, err = redisConfigFinder.Find()
	if err != nil {
		b.logger.Error("redis-config-finder", err, lager.Data{
			"event":     "failed",
			"root_path": b.backupConfig.RedisConfigRoot,
			"file_name": b.backupConfig.RedisConfigFilename,
		})
		return err
	}

	return nil
}

func (b *instanceBackuper) initInstanceIDLocator() error {
	switch b.backupConfig.PlanName {
	case broker.PlanNameShared:
		b.instanceIDLocator = b.sharedInstanceIDLocatorProvider(b.logger)
		return nil
	case broker.PlanNameDedicated:
		b.instanceIDLocator = b.dedicatedInstanceIDLocatorProvider(
			b.backupConfig.BrokerAddress,
			b.backupConfig.BrokerCredentials.Username,
			b.backupConfig.BrokerCredentials.Password,
			b.logger,
		)
		return nil
	}
	return fmt.Errorf("Unknown plan name %s", b.backupConfig.PlanName)
}
