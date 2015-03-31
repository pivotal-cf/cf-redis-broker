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

	backupErrors := []error{}

	backupCreator := backup.Backup{
		Config: config,
		Logger: logger,
	}

	instanceDirs, err := ioutil.ReadDir(config.RedisDataDirectory)

	if config.DedicatedInstance {
		err := backupCreator.Create(config.RedisDataDirectory, config.RedisDataDirectory, config.NodeID, "dedicated-vm")
		if err != nil {
			backupErrors = append(backupErrors, err)
			logger.Error("error backing up dedicated instance", err)
		}
	} else {

		for _, instanceDir := range instanceDirs {

			basename := instanceDir.Name()
			if strings.HasPrefix(basename, ".") {
				continue
			}

			configPath := path.Join(config.RedisDataDirectory, basename)
			instanceDataPath := path.Join(configPath, "db")
			err = backupCreator.Create(configPath, instanceDataPath, basename, "shared-vm")
			if err != nil {
				backupErrors = append(backupErrors, err)
				logger.Error("error backing up instance", err, lager.Data{
					"instance_id": basename,
				})
			}
		}
	}

	if len(backupErrors) > 0 {
		log.Fatal(backupErrors)
	}
}

func configPath() string {
	path := os.Getenv("BACKUP_CONFIG_PATH")
	if path == "" {
		panic("BACKUP_CONFIG_PATH not set")
	}
	return path
}
