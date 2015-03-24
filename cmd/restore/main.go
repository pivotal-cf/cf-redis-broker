package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/pivotal-cf/cf-redis-broker/availability"
	"github.com/pivotal-cf/cf-redis-broker/brokerconfig"
	"github.com/pivotal-cf/cf-redis-broker/process"
	"github.com/pivotal-cf/cf-redis-broker/redis"
	"github.com/pivotal-cf/cf-redis-broker/redis/client"
	"github.com/pivotal-cf/cf-redis-broker/redisconf"
	"github.com/pivotal-cf/cf-redis-broker/system"
	"github.com/pivotal-golang/lager"
)

const pidFileName = "redis-server.pid"
const aofRewriteInProgressCheckIntervalMilliseconds = 100
const monitStatusCheckIntervalMilliseconds = 100
const monitProcessRunningTimeoutMilliseconds = 20000
const monitProcessNotMonitoredTimeoutMilliseconds = 20000
const NumRestoreSteps = 12

func copyRdbFileIntoInstance(rdbPath, instanceDataDirPath string) error {
	source, _ := os.Open(rdbPath)
	defer source.Close()

	destinationPath := filepath.Join(instanceDataDirPath, "dump.rdb")
	destination, err := os.Create(destinationPath)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

var stepCounter int

func startStep(description string) {
	stepCounter++

	dots := ""
	lineLength := 55
	for i := 0; i < (lineLength - len(description)); i++ {
		dots += "."
	}

	fmt.Printf("[%2d/%d] %s%s", stepCounter, NumRestoreSteps, description, dots)
}

func finishStep(status string) {
	fmt.Printf("%s\n", status)
}

func finishStepFatal(description string) {
	fmt.Println("ERROR")
	log.Fatalf(description)
}

func main() {
	fmt.Println("Starting redis restore")

	if len(os.Args) != 3 {
		log.Fatalf("usage: restore <instance_id> <rdb_path>")
	}

	instanceID := os.Args[1]
	rdbPath := os.Args[2]

	logger := lager.NewLogger("redis-restore")

	startStep("Loading config")
	config, err := brokerconfig.ParseConfig(configPath())
	if err != nil {
		finishStepFatal("Could not load config")
	}
	finishStep("OK")

	monitExecutablePath := config.MonitExecutablePath
	instanceDirPath := filepath.Join(config.RedisConfiguration.InstanceDataDirectory, instanceID)
	dataDirPath := filepath.Join(instanceDirPath, "db")

	startStep("Checking instance directory and backup file")
	if _, err := os.Stat(instanceDirPath); os.IsNotExist(err) {
		finishStepFatal("Instance not found")
	}

	if _, err := os.Stat(rdbPath); os.IsNotExist(err) {
		log.Fatalf("RDB file not found")
	}
	finishStep("OK")

	startStep("Copying backup file to instance directory")
	err = copyRdbFileIntoInstance(rdbPath, dataDirPath)
	if err != nil {
		finishStep("ERROR")
		logger.Fatal("copy-rdb", err)
	}
	finishStep("OK")

	localRepo := &redis.LocalRepository{
		RedisConf: config.RedisConfiguration,
	}

	commandRunner := system.OSCommandRunner{
		Logger: logger,
	}

	processController := &redis.OSProcessController{
		CommandRunner:             commandRunner,
		InstanceInformer:          localRepo,
		Logger:                    logger,
		ProcessChecker:            &process.ProcessChecker{},
		ProcessKiller:             &process.ProcessKiller{},
		WaitUntilConnectableFunc:  availability.Check,
		RedisServerExecutablePath: config.RedisServerExecutablePath,
	}

	startStep("Disabling Redis process watcher")
	err = stopProcessWatcher(monitExecutablePath)
	if err != nil {
		finishStep("ERROR")
		logger.Fatal("stop-process-watcher", err)
	}
	finishStep("OK")

	instance := &redis.Instance{ID: instanceID, Host: "localhost", Port: 6379}

	pidfilePath := localRepo.InstancePidFilePath(instanceID)

	startStep("Stopping Redis")
	err = processController.Kill(instance)
	if err != nil {
		finishStep("ERROR")
		logger.Fatal("killing-redis", err)
	}
	finishStep("OK")

	startStep("Starting Redis from backup file")
	err = processController.StartAndWaitUntilReadyWithConfig(
		instance,
		[]string{
			"--pidfile", pidfilePath,
			"--daemonize", "yes",
			"--dir", dataDirPath,
		},
		time.Duration(config.RedisConfiguration.StartRedisTimeoutSeconds)*time.Second,
	)
	if err != nil {
		finishStep("ERROR")
		logger.Fatal("starting-redis", err)
	}
	finishStep("OK")

	startStep("Waiting for redis to finish loading data into memory")
	conf := redisconf.New(
		redisconf.Param{Key: "port", Value: strconv.Itoa(instance.Port)},
		redisconf.Param{Key: "requirepass", Value: instance.Password},
	)
	client, err := client.Connect(instance.Host, conf)
	if err != nil {
		finishStep("ERROR")
		logger.Fatal("connecting-to-redis", err)
	}

	err = client.WaitUntilRedisNotLoading(config.RedisConfiguration.StartRedisTimeoutSeconds * 1000)
	if err != nil {
		finishStep("ERROR")
		logger.Fatal("starting-redis", err)
	}
	finishStep("OK")

	startStep("Enabling appendonly mode")
	err = client.EnableAOF()
	if err != nil {
		finishStep("ERROR")
		logger.Fatal("enabling-aof", err)
	}
	finishStep("OK")

	startStep("Waiting for appendonly rewrite to finish")
	for {
		aofRewriteInProgress, err := client.InfoField("aof_rewrite_in_progress")
		if err != nil {
			finishStep("ERROR")
			logger.Fatal("querying-aof-progress", err)
		}

		if aofRewriteInProgress == "0" {
			break
		}

		time.Sleep(time.Millisecond * aofRewriteInProgressCheckIntervalMilliseconds)
	}
	finishStep("OK")

	startStep("Stopping Redis")
	err = processController.Kill(instance)
	if err != nil {
		finishStep("ERROR")
		logger.Fatal("killing-redis", err)
	}
	finishStep("OK")

	startStep("Setting correct permissions on appendonly file")
	aofPath := path.Join(localRepo.InstanceDataDir(instance.ID), "appendonly.aof")
	err = chownAof("vcap", aofPath)
	if err != nil {
		finishStep("ERROR")
		logger.Fatal("chown-aof", err)
	}
	finishStep("OK")

	startStep("Restarting Redis process watcher")
	err = startProcessWatcher(monitExecutablePath)
	if err != nil {
		finishStep("ERROR")
		logger.Fatal("start-process-watcher", err)
	}
	finishStep("OK")

	fmt.Println("Restore completed successfully")
}

func chownAof(user, aofPath string) error {
	// eg /usr/bin/chown vcap:vcap /tmp/aof.aof
	cmd := exec.Command("chown", fmt.Sprintf("%s:%s", user, user), aofPath)
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func startProcessWatcher(monitExecutablePath string) error {
	_, err := monit(monitExecutablePath, []string{"start", "process-watcher"})
	if err != nil {
		return err
	}

	err = waitUntilMonitStatus(monitExecutablePath, "running", monitProcessRunningTimeoutMilliseconds)
	if err != nil {
		return err
	}

	return nil
}

func stopProcessWatcher(monitExecutablePath string) error {
	_, err := monit(monitExecutablePath, []string{"stop", "process-watcher"})
	if err != nil {
		return err
	}

	err = waitUntilMonitStatus(monitExecutablePath, "not monitored", monitProcessNotMonitoredTimeoutMilliseconds)
	if err != nil {
		return err
	}

	return nil
}

func waitUntilMonitStatus(monitExecutablePath, status string, timeoutMilliseconds int) error {
	timeRemaining := timeoutMilliseconds
	for {
		output, err := monit(monitExecutablePath, []string{"summary"})
		if err != nil {
			return err
		}

		pattern := fmt.Sprintf("Process\\s+'process-watcher'\\s+%s\\n", status)
		matched, err := regexp.MatchString(pattern, output)
		if err != nil {
			return err
		}

		if matched {
			return nil
		}

		time.Sleep(monitStatusCheckIntervalMilliseconds * time.Millisecond)
		timeRemaining -= monitStatusCheckIntervalMilliseconds
		if timeRemaining < 0 {
			return fmt.Errorf("Process process-watcher did not reach '%s' after %d ms", status, timeoutMilliseconds)
		}
	}
}

func monit(executablePath string, args []string) (string, error) {
	cmd := exec.Command(executablePath, args...)
	outputBytes, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(outputBytes), nil
}

func configPath() string {
	brokerConfigYamlPath := os.Getenv("BROKER_CONFIG_PATH")
	if brokerConfigYamlPath == "" {
		return "/var/vcap/jobs/cf-redis-broker/config/broker.yml"
	}
	return brokerConfigYamlPath
}
