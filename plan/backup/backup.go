package backup

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pivotal-cf/cf-redis-broker/broker"
	"github.com/pivotal-cf/cf-redis-broker/plan"
	"github.com/pivotal-cf/cf-redis-broker/plan/dedicated"
	"github.com/pivotal-cf/cf-redis-broker/plan/shared"
	redis "github.com/pivotal-cf/cf-redis-broker/redis/client"
	"github.com/pivotal-golang/lager"
)

type Result struct {
	InstanceID      string
	NodeIP          string
	RedisConfigPath string
	Err             error
}

var defaultInstanceIDProviderFactory = func(config *Config, logger lager.Logger) (plan.IDProvider, error) {
	fmt.Println("Factory: ", config.PlanName)
	switch config.PlanName {
	case broker.PlanNameShared:
		return shared.InstanceIDProvider(logger), nil
	case broker.PlanNameDedicated:
		return dedicated.InstanceIDProvider(
			config.BrokerAddress,
			config.BrokerCredentials.Username,
			config.BrokerCredentials.Password,
			logger,
		), nil
	}
	return nil, fmt.Errorf("Unknown plan name %s", config.PlanName)
}

var defaultRedisConfigLoader = plan.RedisConfigs

var defaultRedisClientProvider = redis.Connect

// var defaultRedisBackupFunc = backup.Backup

var clock = time.Now

func Backup(configPath string, logger lager.Logger) ([]Result, error) {
	logData := lager.Data{
		"backup_config_path": configPath,
	}

	logInfo("starting", logData, logger)

	backupConfig, err := loadBackupConfig(configPath)
	if err != nil {
		logError(err, logData, logger)
		return nil, err
	}

	redisConfigRoot := backupConfig.RedisConfigRoot
	redisConfigFilename := backupConfig.RedisConfigFilename

	redisConfigs, err := defaultRedisConfigLoader(redisConfigRoot, redisConfigFilename)
	if err != nil {
		logError(err, lager.Data{
			"redis_config_root":     redisConfigRoot,
			"redis_config_filename": redisConfigFilename,
		}, logger, "load-redis-configs")
		return nil, err
	}

	idProvider, err := defaultInstanceIDProviderFactory(backupConfig, logger)
	if err != nil {
		logError(err, lager.Data{
			"plan_name": backupConfig.PlanName,
		}, logger, "get-instance-id-provider")
		return nil, err
	}

	// timeout := time.Second * time.Duration(backupConfig.SnapshotTimeoutSeconds)
	results := []Result{}

	for redisConfigPath, redisConfig := range redisConfigs {
		result := Result{}

		// log start instance id
		// instanceID, _ :=
		idProvider.InstanceID(redisConfigPath, backupConfig.NodeIP)
		// check for err
		// log err if happened
		// log end instance id

		//log start redis connect
		_, err := defaultRedisClientProvider(
			redis.Host(redisConfig.Host()),
			redis.Port(redisConfig.Port()),
			redis.Password(redisConfig.Password()),
			redis.CmdAliases(redisConfig.CommandAliases()),
		)
		if err != nil {
			result.Err = err
			results = append(results, result)

			logError(err, lager.Data{
				"address": fmt.Sprintf("%s:%d", redisConfig.Host(), redisConfig.Port()),
			}, logger)

			continue
		}
		//log end redis connect

		// log backup start
		// defaultRedisBackupFunc(
		// 	client,
		// 	timeout,
		// 	backupConfig.S3Config.BucketName,
		// 	targetFilename(backupConfig.S3Config.Path, instanceID, backupConfig.PlanName),
		// 	"s3-endpoint",
		// 	"s3-access-id",
		// 	"s3-secret-key",
		// 	logger,
		// )
		// keep track of backup err
		// log backup err if happened

		//log backup end
		results = append(results, result)

		//disconnect client
	}

	logInfo("done", logData, logger)

	return results, nil
}

func loadBackupConfig(path string) (*Config, error) {
	if path == "" {
		err := errors.New("No backup config path provided")
		return nil, err
	}

	config, err := LoadConfig(path)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func targetFilename(path, instanceID, planName string) string {
	now := clock()
	year := now.Year()
	month := int(now.Month())
	day := now.Day()

	filename := fmt.Sprintf(
		"%s_%s_%s",
		now.Format("200601021504"),
		instanceID,
		planName,
	)

	return fmt.Sprintf("%s/%d/%02d/%02d/%s", path, year, month, day, filename)
}

func logError(err error, data lager.Data, logger lager.Logger, subactions ...string) {
	data["event"] = "failed"
	logger.Error(logAction("plan-backup", subactions), err, data)
}

func logInfo(event string, data lager.Data, logger lager.Logger, subactions ...string) {
	data["event"] = event
	logger.Info(logAction("plan-backup", subactions), data)
}

func logAction(action string, subactions []string) string {
	actions := append([]string{action}, subactions...)
	return strings.Join(actions, ".")
}

func REMOVE_ME_LATER(configPath string, logger lager.Logger) ([]Result, error) {
	if configPath == "" {
		err := errors.New("No backup config path provided")
		logger.Error("instance-backup", err)
		return []Result{}, err
	}

	backupConfig, err := LoadConfig(configPath)
	if err != nil {
		logger.Error(
			"instance-backup",
			err,
			lager.Data{
				"config_path": configPath,
			},
		)
		return []Result{}, err
	}

	configRoot := backupConfig.RedisConfigRoot
	configFilename := backupConfig.RedisConfigFilename

	// timeout := time.Second
	redisConfigs, err := plan.RedisConfigs(configRoot, configFilename)
	if err != nil {
		return []Result{}, err
	}

	var iidProvider plan.IDProvider

	results := []Result{}

	for redisConfigPath, redisConfig := range redisConfigs {
		result := Result{
			NodeIP:          backupConfig.NodeIP,
			RedisConfigPath: redisConfigPath,
		}

		iid, err := iidProvider.InstanceID(redisConfigPath, backupConfig.NodeIP)
		if err != nil {
			result.Err = err
			results = append(results, result)
			continue
		}

		result.InstanceID = iid

		client, err := redis.Connect(
			redis.Host(redisConfig.Host()),
			redis.Port(redisConfig.Port()),
			redis.Password(redisConfig.Password()),
			redis.CmdAliases(redisConfig.CommandAliases()),
		)
		if err != nil {
			result.Err = err
			results = append(results, result)
			continue
		}

		s3 := backupConfig.S3Config

		now := time.Now()
		year := now.Year()
		month := int(now.Month())
		day := now.Day()

		filename := fmt.Sprintf(
			"%s_%s_%s",
			now.Format("200601021504"),
			iid,
			backupConfig.PlanName,
		)

		fmt.Sprintf("%s/%d/%d/%d/%s", s3.Path, year, month, day, filename)

		// err = backup.Backup(
		// 	client,
		// 	timeout,
		// 	s3.BucketName,
		// 	targetPath,
		// 	s3.EndpointUrl,
		// 	s3.AccessKeyId,
		// 	s3.SecretAccessKey,
		// 	logger,
		// )
		if err != nil {
			result.Err = err
			results = append(results, result)
		}

		if err := client.Disconnect(); err != nil {
			// log
		}
	}

	return results, nil
}
