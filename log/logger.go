package log

import (
	"fmt"
	"log"
	"os"

	"github.com/pivotal-cf/cf-redis-broker/backupconfig"
	"github.com/pivotal-golang/lager"
)

type LoggerWithOutput struct {
	lager.Logger
}

var logger *LoggerWithOutput

func Logger() LoggerWithOutput {
	if logger == nil {
		fmt.Println("Logger not initialized, initializing a new one")
		newLager := lager.NewLogger("redis-broker")
		logger = &LoggerWithOutput{newLager}
	}

	return *logger
}

func (l LoggerWithOutput) Info(action string, data ...lager.Data) {
	var outputData lager.Data
	if len(data) > 0 {
		outputData = data[0]
	}

	l.printToOutput(action, outputData)
	l.Logger.Info(action, data...)
}

func (l LoggerWithOutput) printToOutput(action string, data lager.Data) {
	if data != nil && data["event"] != nil {
		fmt.Printf("%15s -> %s\n", action, data["event"])
	}
}

func SetupLogger(config *backupconfig.Config) {
	if logger == nil {
		lagerLogger := initializeLagerLogger(config)
		logger = &LoggerWithOutput{lagerLogger}
	}
}

func initializeLagerLogger(config *backupconfig.Config) lager.Logger {
	logger := lager.NewLogger("backup")
	logFile, err := os.OpenFile(config.LogFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0660)
	if err != nil {
		log.Fatal("unable to open log file")
	}
	logger.RegisterSink(lager.NewWriterSink(logFile, lager.INFO))
	return logger
}
