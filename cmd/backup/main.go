package main

import (
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/pivotal-cf/cf-redis-broker/backup"
	"github.com/pivotal-cf/cf-redis-broker/backupconfig"
	"github.com/pivotal-golang/lager"
)

func main() {
	logger := lager.NewLogger("backup")

	configPath := os.Getenv("BACKUP_CONFIG_PATH")
	if configPath == "" {
		logger.Fatal("BACKUP_CONFIG_PATH not set", nil)
	}

	config, err := backupconfig.Load(configPath)
	if err != nil {
		logger.Fatal("backup-config-load-failed", err)
	}

	if config.S3Configuration.BucketName == "" || config.S3Configuration.EndpointUrl == "" {
		logger.Info("s3 credentials not configured")
		os.Exit(0)
	}

	backupCreator := &backup.Backup{
		Config: config,
		Logger: logger,
	}
	backupErrors := map[string]error{}

	if config.DedicatedInstance {
		if err := backupCreator.Create(config.RedisDataDirectory, "", config.NodeID, "dedicated-vm"); err != nil {
			backupErrors[config.NodeID] = err
		}
	} else {
		backupErrors = backupSharedVMInstances(backupCreator, config.RedisDataDirectory)
	}

	if len(backupErrors) > 0 {
		logBackupErrors(backupErrors, logger)
		os.Exit(1)
	}
}

func backupSharedVMInstances(backupCreator *backup.Backup, instancesDir string) map[string]error {
	instanceDirs, err := ioutil.ReadDir(instancesDir)
	if err != nil {
		return map[string]error{"all-shared-vm-instances": err}
	}

	errors := map[string]error{}
	for _, instanceDir := range instanceDirs {
		basename := instanceDir.Name()
		if strings.HasPrefix(basename, ".") {
			continue
		}

		if err := backupCreator.Create(path.Join(instancesDir, basename), "db", basename, "shared-vm"); err != nil {
			errors[basename] = err
		}
	}
	return errors
}

func logBackupErrors(errors map[string]error, logger lager.Logger) {
	for instanceID, err := range errors {
		logger.Error("backup-failed", err, lager.Data{
			"instance_id": instanceID,
		})
	}
}
