package main

import (
	"log"
	"os"

	"github.com/pivotal-cf/cf-redis-broker/backup"
	"github.com/pivotal-cf/cf-redis-broker/brokerconfig"
	"github.com/pivotal-cf/cf-redis-broker/redis"
	"github.com/pivotal-golang/lager"
)

func main() {
	logger := lager.NewLogger("backup")

	config, err := brokerconfig.ParseConfig(configPath())
	if err != nil {
		log.Fatal(err)
	}

	if !config.RedisConfiguration.BackupConfiguration.Enabled() {
		return
	}

	localRepo := &redis.LocalRepository{
		RedisConf: config.RedisConfiguration,
	}

	instances, err := localRepo.AllInstances()
	if err != nil {
		log.Fatal(err)
	}

	backupErrors := []error{}

	backupCreator := backup.Backup{
		Config: &config,
		Logger: logger,
	}

	for _, instance := range instances {
		instancePath := localRepo.InstanceBaseDir(instance.ID)
		err = backupCreator.Create(instancePath, instance.ID)
		if err != nil {
			backupErrors = append(backupErrors, err)
			logger.Error("error backing up instance", err, lager.Data{
				"instance_id": instance.ID,
			})
		}
	}

	if len(backupErrors) > 0 {
		log.Fatal(backupErrors)
	}
}

func configPath() string {
	brokerConfigYamlPath := os.Getenv("BROKER_CONFIG_PATH")
	if brokerConfigYamlPath == "" {
		panic("BROKER_CONFIG_PATH not set")
	}
	return brokerConfigYamlPath
}
