package main

import (
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/pivotal-cf/cf-redis-broker/availability"
	"github.com/pivotal-cf/cf-redis-broker/brokerconfig"
	"github.com/pivotal-cf/cf-redis-broker/process"
	"github.com/pivotal-cf/cf-redis-broker/redis"
	"github.com/pivotal-cf/cf-redis-broker/system"
	"github.com/pivotal-golang/lager"
)

func main() {
	logger := lager.NewLogger("process-monitor")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))
	logger.RegisterSink(lager.NewWriterSink(os.Stderr, lager.ERROR))

	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, syscall.SIGUSR1)
	skipProcessCheck := false
	go func() {
		<-sigChannel
		logger.Info("Trapped USR1, disabling process monitor")
		skipProcessCheck = true
	}()

	config, err := brokerconfig.ParseConfig(configPath())
	if err != nil {
		logger.Fatal("could not parse config file", err, lager.Data{
			"config-path": configPath(),
		})
	}

	logger.Info("Starting process monitor")

	repo := redis.NewLocalRepository(config.RedisConfiguration, logger)

	commandRunner := system.OSCommandRunner{
		Logger: logger,
	}

	processController := &redis.OSProcessController{
		Logger:                   logger,
		InstanceInformer:         repo,
		CommandRunner:            commandRunner,
		ProcessChecker:           &process.ProcessChecker{},
		ProcessKiller:            &process.ProcessKiller{},
		PingFunc:                 redis.PingServer,
		WaitUntilConnectableFunc: availability.Check,
	}

	checkInterval := config.RedisConfiguration.ProcessCheckIntervalSeconds

	instances, _ := repo.AllInstancesVerbose()

	for _, instance := range instances {
		copyConfigFile(instance, repo, logger)
	}

	for {
		if skipProcessCheck {
			logger.Info("Skipping instance check")
		} else {
			instances, _ := repo.AllInstances()

			for _, instance := range instances {
				ensureRunningIfNotLocked(instance, repo, processController, logger)
			}
		}

		time.Sleep(time.Second * time.Duration(checkInterval))
	}
}

func ensureRunningIfNotLocked(instance *redis.Instance, repo *redis.LocalRepository, processController *redis.OSProcessController, logger lager.Logger) {
	_, err := os.Stat(filepath.Join(repo.InstanceBaseDir(instance.ID), "lock"))
	if err != nil {
		ensureRunning(instance, repo, processController, logger)
	}
}

func copyConfigFile(instance *redis.Instance, repo *redis.LocalRepository, logger lager.Logger) {
	err := repo.EnsureDirectoriesExist(instance)
	if err != nil {
		logger.Fatal("Error creating instance directories", err, lager.Data{
			"instance": instance.ID,
		})
	}

	err = repo.WriteConfigFile(instance)
	if err != nil {
		logger.Fatal("Error writing redis config", err, lager.Data{
			"instance": instance.ID,
		})
	}
}

func ensureRunning(instance *redis.Instance, repo *redis.LocalRepository, processController *redis.OSProcessController, logger lager.Logger) {
	configPath := repo.InstanceConfigPath(instance.ID)
	instanceDataDir := repo.InstanceDataDir(instance.ID)
	pidfilePath := repo.InstancePidFilePath(instance.ID)
	logfilePath := repo.InstanceLogFilePath(instance.ID)

	err := processController.EnsureRunning(instance, configPath, instanceDataDir, pidfilePath, logfilePath)
	if err != nil {
		logger.Fatal("Error starting instance", err, lager.Data{
			"instance": instance.ID,
		})
	}
}

func configPath() string {
	brokerConfigYamlPath := os.Getenv("BROKER_CONFIG_PATH")
	if brokerConfigYamlPath == "" {
		panic("BROKER_CONFIG_PATH not set")
	}
	return brokerConfigYamlPath
}
