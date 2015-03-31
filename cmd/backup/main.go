package main

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"

	"github.com/pivotal-cf/cf-redis-broker/backup"
	"github.com/pivotal-cf/cf-redis-broker/backupconfig"
	"github.com/pivotal-golang/lager"
)

func main() {
	logger := lager.NewLogger("backup")

	config, err := backupconfig.Load(configPath())
	if err != nil {
		log.Fatal(err)
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
		for instanceID, err := range backupErrors {
			logger.Error("backup-failed", err, lager.Data{
				"instance_id": instanceID,
			})
		}
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

func configPath() string {
	path := os.Getenv("BACKUP_CONFIG_PATH")
	if path == "" {
		panic("BACKUP_CONFIG_PATH not set")
	}
	return path
}
