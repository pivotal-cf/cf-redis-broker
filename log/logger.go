package log

import (
	"log"
	"os"

	"github.com/pivotal-cf/cf-redis-broker/backupconfig"
	"github.com/pivotal-golang/lager"
)

var logger lager.Logger

func Logger() lager.Logger {
	if logger == nil {
		logger = lager.NewLogger("redis-broker")
	}
	return logger
}

func SetupLogger(config *backupconfig.Config) {
	if logger == nil {
		logger = initializeLagerLogger(config)
	}
}

func initializeLagerLogger(config *backupconfig.Config) lager.Logger {
	logger := lager.NewLogger("backup")
	logFile, err := os.OpenFile(config.LogFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0660)
	if err != nil {
		log.Fatal("unable to open log file")
	}
	logger.RegisterSink(lager.NewWriterSink(logFile, lager.INFO))
	logger.RegisterSink(NewCliSink(lager.INFO))
	return logger
}
