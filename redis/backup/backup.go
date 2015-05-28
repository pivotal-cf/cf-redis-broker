package backup

import (
	"fmt"

	"code.google.com/p/go-uuid/uuid"
	"github.com/pivotal-cf/cf-redis-broker/recovery/task"
	redis "github.com/pivotal-cf/cf-redis-broker/redis/client"
	"github.com/pivotal-golang/lager"
)

func Backup(client redis.Client, logger lager.Logger) error {
	snapshot := NewSnapshot(client, 123, logger)
	img, err := snapshot.Create()
	if err != nil {
		fmt.Println("Snapshot failed: ", err.Error())
	}

	originalPath := img.Path()
	tmpSnapshotPath := uuid.New()

	img, err = task.NewPipeline(
		"redis-backup",
		logger,
		task.NewRename(tmpSnapshotPath, logger),
		task.NewS3Upload("bucket-name", "target-path", "endpoint", "key", "secret", logger),
	).Run(img)

	task.NewPipeline(
		"cleanup",
		logger,
		NewCleanup(
			originalPath,
			tmpSnapshotPath,
			logger,
		),
	).Run(img)

	return err
}
