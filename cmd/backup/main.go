package main

import (
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/pivotal-cf/cf-redis-broker/brokerconfig"
	"github.com/pivotal-cf/cf-redis-broker/redis"
	"github.com/pivotal-golang/lager"

	cf_lager "github.com/cloudfoundry-incubator/cf-lager"
)

var logger = cf_lager.New("backup")

func main() {
	config, err := brokerconfig.ParseConfig(configPath())
	if err != nil {
		log.Fatal(err)
	}

	if !config.RedisConfiguration.BackupConfiguration.Enabled() {
		return
	}

	instanceDirs, err := ioutil.ReadDir(config.RedisConfiguration.InstanceDataDirectory)
	if err != nil {
		log.Fatal(err)
	}

	backupErrors := []error{}
	for _, instanceDir := range instanceDirs {
		if strings.HasPrefix(instanceDir.Name(), ".") {
			continue
		}

		backup := redis.Backup{
			Config: &config,
		}

		err = backup.Create(instanceDir.Name())
		if err != nil {
			backupErrors = append(backupErrors, err)
			logger.Error("error backing up instance", err, lager.Data{
				"instance_id": instanceDir.Name(),
			})
		}

	}

	if len(backupErrors) > 0 {
		os.Exit(1)
	}
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func configPath() string {
	brokerConfigYamlPath := os.Getenv("BROKER_CONFIG_PATH")
	if brokerConfigYamlPath == "" {
		panic("BROKER_CONFIG_PATH not set")
	}
	return brokerConfigYamlPath
}
