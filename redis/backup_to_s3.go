package redis

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

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
	logger := lager.NewLogger("backup")

	bucket := backup.createBucket()

	if !backup.validateInstanceDirectoryIsPresentFor(instanceID, logger) {
		return nil
	}

	err := backup.createSnapshot(instanceID)
	if err != nil {
		return err
	}

	pathToRdbFile := filepath.Join(backup.Config.RedisConfiguration.InstanceDataDirectory, instanceID, "db", "dump.rdb")

	if !backup.validateBackupFileCreatedFor(pathToRdbFile, logger) {
		return nil
	}

	return backup.uploadToS3(instanceID, pathToRdbFile, bucket, logger)
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func (backup Backup) createSnapshot(instanceID string) error {
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

func (backup Backup) validateInstanceDirectoryIsPresentFor(instanceID string, logger lager.Logger) bool {
	pathToInstanceDirectory := filepath.Join(backup.Config.RedisConfiguration.InstanceDataDirectory, instanceID)
	if !fileExists(pathToInstanceDirectory) {
		logger.Info("instance directory not found, skipping instance backup", lager.Data{
			"Local file": pathToInstanceDirectory,
		})
		return false
	}
	return true
}

func (backup Backup) validateBackupFileCreatedFor(pathToRdbFile string, logger lager.Logger) bool {
	if !fileExists(pathToRdbFile) {
		logger.Info("dump.rb not found, skipping instance backup", lager.Data{
			"Local file": pathToRdbFile,
		})
		return false
	}
	return true
}

func (backup Backup) uploadToS3(instanceID, pathToRdbFile string, bucket s3bucket.Bucket, logger lager.Logger) error {
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

func (backup Backup) createBucket() s3bucket.Bucket {
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
	return bucket
}
