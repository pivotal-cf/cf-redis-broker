package backup

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/pivotal-cf/cf-redis-broker/backup/s3bucket"
	"github.com/pivotal-cf/cf-redis-broker/backupconfig"
	"github.com/pivotal-cf/cf-redis-broker/redis/client"
	"github.com/pivotal-cf/cf-redis-broker/redisconf"
	"github.com/pivotal-golang/lager"
)

type Backup struct {
	Config *backupconfig.Config
	Logger lager.Logger
}

// http://golang.org/pkg/time/#pkg-constants if you need to understand this crazy layout
const (
	filenameTimestampLayout = "200601021504"
	datePathLayout          = "2006/01/02"
)

func (backup Backup) Create(instancePath, dataSubDir, instanceID, planName string) error {
	if err := backup.createSnapshot(instancePath); err != nil {
		return err
	}

	timestamp := time.Now().Format(filenameTimestampLayout)

	rdbFilePath := filepath.Join(instancePath, dataSubDir, "dump.rdb")
	if !fileExists(rdbFilePath) {
		backup.Logger.Info("dump.rdb not found, skipping instance backup", lager.Data{
			"Local file": rdbFilePath,
		})
		return nil
	}

	bucket, err := backup.getOrCreateBucket()
	if err != nil {
		return err
	}
	return backup.uploadToS3(instanceID, planName, rdbFilePath, timestamp, bucket)
}

func (backup Backup) getOrCreateBucket() (s3bucket.Bucket, error) {
	s3Client := s3bucket.NewClient(
		backup.Config.S3Configuration.EndpointUrl,
		backup.Config.S3Configuration.AccessKeyId,
		backup.Config.S3Configuration.SecretAccessKey,
	)

	return s3Client.GetOrCreate(backup.Config.S3Configuration.BucketName)
}

func (backup Backup) createSnapshot(instancePath string) error {
	instanceConf, err := redisconf.Load(path.Join(instancePath, "redis.conf"))
	if err != nil {
		return err
	}

	client, err := client.Connect("localhost", instanceConf)
	if err != nil {
		return err
	}

	return client.CreateSnapshot(backup.Config.BGSaveTimeoutSeconds)
}

func (backup Backup) uploadToS3(instanceID, planName, rdbFilePath string, timestamp string, bucket s3bucket.Bucket) error {
	rdbBytes, err := ioutil.ReadFile(rdbFilePath)
	if err != nil {
		return err
	}

	remotePath := fmt.Sprintf("%s/%s/%s_%s_%s_redis_backup",
		backup.Config.S3Configuration.Path,
		time.Now().Format(datePathLayout),
		timestamp,
		instanceID,
		planName,
	)

	backup.Logger.Info("Backing up instance", lager.Data{
		"Local file":  rdbFilePath,
		"Remote file": remotePath,
	})

	return bucket.Upload(rdbBytes, remotePath)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}
