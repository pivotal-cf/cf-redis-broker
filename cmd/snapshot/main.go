package main

import (
	"flag"
	"os"

	"github.com/pivotal-cf/cf-redis-broker/instance/backup"
	"github.com/pivotal-cf/cf-redis-broker/log"
	"github.com/pivotal-golang/lager"
)

func main() {
	logger := lager.NewLogger("backup")
	logger.RegisterSink(log.NewCliSink(lager.INFO))
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.INFO))

	var configPath string
	flag.StringVar(&configPath, "config", "", "Path to YML config file")

	flag.Parse()

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		logger.Error("snapshot-main", err, lager.Data{"event": "failed", "config_file": configPath})
		os.Exit(2)
	}

	backupConfig, _ := backup.LoadBackupConfig(configPath)

	logFile, err := os.OpenFile(backupConfig.LogFilepath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0660)
	if err != nil {
		logger.Error("snapshot-main", err, lager.Data{"event": "failed", "LogFilepath": backupConfig.LogFilepath})
	}
	logger.RegisterSink(lager.NewWriterSink(logFile, lager.INFO))

	//	TODO: handle error

	// TODO: Should this really be a pointer not a struct?
	if (backupConfig.S3Config == backup.S3Configuration{}) {
		logger.Info("snapshot-main", lager.Data{"event": "cancelled", "message": "S3 configuration not found - skipping backup"})
		os.Exit(0)
	}

	backuper, _ := backup.NewInstanceBackuper(*backupConfig, logger)

	backuper.Backup()

	/*client, err := redis.Connect()
	if err != nil {
		logger.Error(
			"backup_main",
			err,
			lager.Data{
				"event": "redis_connection_failed",
			},
		)
		os.Exit(1)
	}

	redisBackuper := backup.NewRedisBackuper(nil, nil, nil, nil, nil, nil, nil)

	if err := backup.Backup(client, logger); err != nil {
		logger.Error(
			"backup_main",
			err,
			lager.Data{
				"event": "redis_backup_failed",
			},
		)
		os.Exit(1)
	}*/
}
