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
