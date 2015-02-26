package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
	"github.com/pivotal-cf/cf-redis-broker/brokerconfig"
	"github.com/pivotal-cf/cf-redis-broker/redis"
	"github.com/pivotal-cf/cf-redis-broker/redis/client"
	"github.com/pivotal-cf/cf-redis-broker/redisconf"
	"github.com/pivotal-golang/lager"

	cf_lager "github.com/cloudfoundry-incubator/cf-lager"
)

var S3RegionNotFoundErr error = errors.New("S3 region not found")
var logger = cf_lager.New("backup")

func main() {
	config := parseConfig()
	if !config.RedisConfiguration.BackupConfiguration.Enabled() {
		return
	}

	bucket, err := getBucket(config.RedisConfiguration.BackupConfiguration)
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

func parseConfig() brokerconfig.Config {
	config, err := brokerconfig.ParseConfig(configPath())
	if err != nil {
		log.Fatal(err)
	}
	return config
}

func backupInstance(instanceDir os.FileInfo, config brokerconfig.Config, bucket *s3.Bucket) error {

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

	return bucket.Put(remotePath, rdbBytes, "", "")
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func getBucket(backupConfig brokerconfig.BackupConfiguration) (*s3.Bucket, error) {
	client, err := configureS3Client(backupConfig)
	if err != nil {
		return nil, err
	}
	bucket := client.Bucket(backupConfig.BucketName)

	foundBucketOnS3 := false

	resp, err := client.ListBuckets()
	if err != nil {
		return nil, err
	}

	for _, bucket := range resp.Buckets {
		if bucket.Name == backupConfig.BucketName {
			foundBucketOnS3 = true
			logger.Info("Bucket exists", lager.Data{"bucket": backupConfig.BucketName})
			break
		}
	}

	if !foundBucketOnS3 {
		logger.Info("Creating bucket", lager.Data{"bucket": backupConfig.BucketName})
		err = bucket.PutBucket(s3.Private)
		if err != nil {
			return nil, err
		}
	}

	return bucket, nil
}

func saveAndWaitUntilFinished(instanceDir os.FileInfo, config brokerconfig.Config) error {

	instanceID := instanceDir.Name()

	client, err := buildRedisClient(instanceID, config)
	if err != nil {
		return err
	}

	lastSaveTime, err := client.LastRDBSaveTime()
	if err != nil {
		return err
	}

	// ensure new save time can't match last save time
	time.Sleep(time.Second)

	err = client.RunBGSave()
	if err != nil {
		return err
	}

	// wait for save to complete
	timeout := config.RedisConfiguration.BackupConfiguration.BGSaveTimeoutSeconds
	for i := 0; i < timeout; i++ {
		saveTime, err := client.LastRDBSaveTime()
		if err != nil {
			return err
		}

		if saveTime > lastSaveTime {
			return nil
		}

		time.Sleep(time.Second)
	}

	return errors.New("Timed out waiting for background save to complete")
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

func configureS3Client(backupConfig brokerconfig.BackupConfiguration) (*s3.S3, error) {
	// Warning! Fake S3 server does not appear to honour region constraints or authentication.
	// The fact that this uses the configured region and authentication is untested.
	region, err := getRegion(backupConfig)
	if err != nil {
		return nil, err
	}

	auth := aws.Auth{
		AccessKey: backupConfig.AccessKeyId,
		SecretKey: backupConfig.SecretAccessKey,
	}

	return s3.New(auth, region), nil
}

func getRegion(backupConfig brokerconfig.BackupConfiguration) (aws.Region, error) {
	for _, region := range aws.Regions {
		if region.S3Endpoint == backupConfig.EndpointUrl {
			return region, nil
		}
	}

	if backupConfig.S3Region != "" {
		return aws.Region{
			Name:                 backupConfig.S3Region,
			S3Endpoint:           backupConfig.EndpointUrl,
			S3LocationConstraint: true,
		}, nil
	}

	return aws.Region{}, S3RegionNotFoundErr
}

func configPath() string {
	brokerConfigYamlPath := os.Getenv("BROKER_CONFIG_PATH")
	if brokerConfigYamlPath == "" {
		panic("BROKER_CONFIG_PATH not set")
	}
	return brokerConfigYamlPath
}
