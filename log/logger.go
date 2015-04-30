package log

import (
	"fmt"
	"log"
	"os"

	"github.com/pivotal-cf/cf-redis-broker/backupconfig"
	"github.com/pivotal-golang/lager"
)

var logger lager.Logger

func Logger() lager.Logger {
	if logger == nil {
		fmt.Println("Logger not initialized, initializing a new one")
		logger = lager.NewLogger("redis-broker")
	}

	return logger
}

func SetupLogger(config *backupconfig.Config) {
	if logger == nil {
		logger = initializeLogger(config)
	}
}

func initializeLogger(config *backupconfig.Config) lager.Logger {
	logger := lager.NewLogger("backup")
	logFile, err := os.OpenFile(config.LogFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0660)
	if err != nil {
		log.Fatal("unable to open log file")
	}
	logger.RegisterSink(lager.NewWriterSink(logFile, lager.INFO))
	return logger
}
