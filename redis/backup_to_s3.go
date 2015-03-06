package redis

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/cloudfoundry-incubator/cf-lager"
	"github.com/pivotal-cf/cf-redis-broker/brokerconfig"
	"github.com/pivotal-cf/cf-redis-broker/redis/client"
	"github.com/pivotal-cf/cf-redis-broker/redisconf"
	"github.com/pivotal-cf/cf-redis-broker/s3bucket"
	"github.com/pivotal-golang/lager"
)

type Backup struct {
	Config *brokerconfig.Config
}

func (backup Backup) Create(instanceID string) error {
	s3Client := s3bucket.NewClient(
		backup.Config.RedisConfiguration.BackupConfiguration.EndpointUrl,
		backup.Config.RedisConfiguration.BackupConfiguration.S3Region,
		backup.Config.RedisConfiguration.BackupConfiguration.AccessKeyId,
		backup.Config.RedisConfiguration.BackupConfiguration.SecretAccessKey,
	)

	bucket, err := s3Client.GetOrCreate(backup.Config.RedisConfiguration.BackupConfiguration.BucketName)
	if err != nil {
		log.Fatal(err)
	}

	return backup.backupInstance(instanceID, bucket)
}

func (backup Backup) backupInstance(instanceID string, bucket s3bucket.Bucket) error {
	logger := cf_lager.New("backup")

	pathToInstanceDirectory := filepath.Join(backup.Config.RedisConfiguration.InstanceDataDirectory, instanceID)
	if !fileExists(pathToInstanceDirectory) {
		logger.Info("instance directory not found, skipping instance backup", lager.Data{
			"Local file": pathToInstanceDirectory,
		})
		return nil
	}

	err := backup.saveAndWaitUntilFinished(instanceID)
	if err != nil {
		return err
	}

	pathToRdbFile := filepath.Join(backup.Config.RedisConfiguration.InstanceDataDirectory, instanceID, "db", "dump.rdb")
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

	remotePath := fmt.Sprintf("%s/%s", backup.Config.RedisConfiguration.BackupConfiguration.Path, instanceID)

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

func (backup Backup) saveAndWaitUntilFinished(instanceID string) error {
	client, err := backup.buildRedisClient(instanceID)
	if err != nil {
		return err
	}

	return client.CreateSnapshot(backup.Config.RedisConfiguration.BackupConfiguration.BGSaveTimeoutSeconds)
}

func (backup Backup) buildRedisClient(instanceID string) (*client.Client, error) {

	localRepo := LocalRepository{RedisConf: backup.Config.RedisConfiguration}
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
