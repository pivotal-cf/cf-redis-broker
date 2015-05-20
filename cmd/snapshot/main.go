package main

import (
	"os"

	"github.com/pivotal-cf/cf-redis-broker/log"
	"github.com/pivotal-cf/cf-redis-broker/redis/backup"
	redis "github.com/pivotal-cf/cf-redis-broker/redis/client"
	"github.com/pivotal-golang/lager"
)

func main() {
	logger := lager.NewLogger("backup")
	logger.RegisterSink(log.NewCliSink(lager.INFO))

	client, err := redis.Connect()
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

	if err := backup.Backup(client, logger); err != nil {
		logger.Error(
			"backup_main",
			err,
			lager.Data{
				"event": "redis_backup_failed",
			},
		)
		os.Exit(1)
	}
}
