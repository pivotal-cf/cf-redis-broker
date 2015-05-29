package backup

import (
	"io/ioutil"
	"path/filepath"
	"time"

	"code.google.com/p/go-uuid/uuid"
	"github.com/pivotal-cf/cf-redis-broker/recovery/task"
	redis "github.com/pivotal-cf/cf-redis-broker/redis/client"
	"github.com/pivotal-golang/lager"
)

var snapshotProvider = NewSnapshot
var renameProvider = task.NewRename
var s3UploadProvider = task.NewS3Upload
var cleanupProvider = NewCleanup

func Backup(
	client redis.Client,
	snapshotTimeout time.Duration,
	s3BucketName string,
	s3TargetPath string,
	s3Endpoint string,
	awsAccessKey string,
	awsSecretKey string,
	logger lager.Logger,
) error {
	localLogger := logger.WithData(lager.Data{
		"redis_address": client.Address(),
	})

	localLogger.Info("backup", lager.Data{"event": "starting"})

	snapshot := snapshotProvider(client, snapshotTimeout, logger)
	artifact, err := snapshot.Create()
	if err != nil {
		localLogger.Error("backup", err, lager.Data{"event": "failed"})
		return err
	}

	originalPath := artifact.Path()
	tmpDir, err := ioutil.TempDir("", "redis-backup")
	if err != nil {
		localLogger.Error("backup", err, lager.Data{"event": "failed"})
		return err
	}

	tmpSnapshotPath := filepath.Join(tmpDir, uuid.New())

	artifact, err = task.NewPipeline(
		"redis-backup",
		logger,
		renameProvider(tmpSnapshotPath, logger),
		s3UploadProvider(s3BucketName, s3TargetPath, s3Endpoint, awsAccessKey, awsSecretKey, logger),
	).Run(artifact)

	if err != nil {
		localLogger.Error("backup", err, lager.Data{"event": "failed"})
	}

	task.NewPipeline(
		"cleanup",
		logger,
		cleanupProvider(
			originalPath,
			tmpSnapshotPath,
			logger,
		),
	).Run(artifact)

	localLogger.Info("backup", lager.Data{"event": "done"})

	return err
}
