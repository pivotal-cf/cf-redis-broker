package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/pivotal-cf/cf-redis-broker/brokerconfig"
	"github.com/pivotal-cf/cf-redis-broker/redis"
	"github.com/pivotal-cf/cf-redis-broker/redis/client"
	"github.com/pivotal-cf/cf-redis-broker/redisconf"
	"github.com/pivotal-cf/cf-redis-broker/s3bucket"
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

	s3Client := s3bucket.NewClient(
		config.RedisConfiguration.BackupConfiguration.EndpointUrl,
		config.RedisConfiguration.BackupConfiguration.S3Region,
		config.RedisConfiguration.BackupConfiguration.AccessKeyId,
		config.RedisConfiguration.BackupConfiguration.SecretAccessKey,
	)

	bucket, err := s3Client.GetOrCreate(config.RedisConfiguration.BackupConfiguration.BucketName)
	if err != nil {
		log.Fatal(err)
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
		err = backupInstance(instanceDir, config, bucket)
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

func backupInstance(instanceDir os.FileInfo, config brokerconfig.Config, bucket s3bucket.Bucket) error {
	pathToInstanceDirectory := filepath.Join(config.RedisConfiguration.InstanceDataDirectory, instanceDir.Name())
	if !fileExists(pathToInstanceDirectory) {
		logger.Info("instance directory not found, skipping instance backup", lager.Data{
			"Local file": pathToInstanceDirectory,
		})
		return nil
	}

	err := saveAndWaitUntilFinished(instanceDir, config)
	if err != nil {
		return err
	}

	pathToRdbFile := filepath.Join(config.RedisConfiguration.InstanceDataDirectory, instanceDir.Name(), "db", "dump.rdb")
	if !fileExists(pathToRdbFile) {
		logger.Info("dump.rb not found, skipping instance backup", lager.Data{
			"Local file": pathToRdbFile,
		})
		return nil
	}

	rdbBytes, err := ioutil.ReadFile(pathToRdbFile)
	if err != nil {
		return err
	}

	remotePath := fmt.Sprintf("%s/%s", config.RedisConfiguration.BackupConfiguration.Path, instanceDir.Name())

	logger.Info("Backing up instance", lager.Data{
		"Local file":  pathToRdbFile,
		"Remote file": remotePath,
	})

	return bucket.Upload(rdbBytes, remotePath)
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func saveAndWaitUntilFinished(instanceDir os.FileInfo, config brokerconfig.Config) error {

	instanceID := instanceDir.Name()

	client, err := buildRedisClient(instanceID, config)
	if err != nil {
		return err
	}

	return client.CreateSnapshot(config.RedisConfiguration.BackupConfiguration.BGSaveTimeoutSeconds)
}

func buildRedisClient(instanceID string, config brokerconfig.Config) (*client.Client, error) {

	localRepo := redis.LocalRepository{RedisConf: config.RedisConfiguration}
	instance, err := localRepo.FindByID(instanceID)
	if err != nil {
		return nil, err
	}

	instanceConf, err := redisconf.Load(localRepo.InstanceConfigPath(instanceID))
	if err != nil {
		return nil, err
	}

	return client.Connect(instance.Host, uint(instance.Port), instance.Password, instanceConf)
}

func configPath() string {
	brokerConfigYamlPath := os.Getenv("BROKER_CONFIG_PATH")
	if brokerConfigYamlPath == "" {
		panic("BROKER_CONFIG_PATH not set")
	}
	return brokerConfigYamlPath
}
