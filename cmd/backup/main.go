package main

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
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

	if isDedicatedInstance(instanceDirs) {
		instanceDataPath := filepath.Join(config.RedisDataDirectory)
		configPath := filepath.Join(config.RedisDataDirectory)

		err := backupCreator.Create(configPath, instanceDataPath, config.NodeID)
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

			instanceDataPath := path.Join(config.RedisDataDirectory, basename, "db")
			configPath := path.Join(config.RedisDataDirectory, basename)
			err = backupCreator.Create(configPath, instanceDataPath, basename)
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

func isDedicatedInstance(instanceDirs []os.FileInfo) bool {
	dedicatedInstance := sort.Search(len(instanceDirs), func(i int) bool {
		return instanceDirs[i].Name() == "redis.conf"
	})
	return dedicatedInstance < len(instanceDirs)
}
