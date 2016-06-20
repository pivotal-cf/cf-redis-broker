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
	"time"

	"github.com/pivotal-cf/cf-redis-broker/availability"
	"github.com/pivotal-cf/cf-redis-broker/process"
	"github.com/pivotal-cf/cf-redis-broker/redis"
	"github.com/pivotal-cf/cf-redis-broker/redis/client"
	"github.com/pivotal-cf/cf-redis-broker/redisconf"
	"github.com/pivotal-cf/cf-redis-broker/restoreconfig"
	"github.com/pivotal-cf/cf-redis-broker/system"
	"github.com/pivotal-golang/lager"
)

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
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))
	logger.RegisterSink(lager.NewWriterSink(os.Stderr, lager.ERROR))

	startStep("Loading config")
	config, err := restoreconfig.Load(restoreConfigPath())
	if err != nil {
		finishStepFatal("Could not load config")
	}
	finishStep("OK")

	monitExecutablePath := config.MonitExecutablePath
	instanceDirPath := config.InstanceDataDir(instanceID)

	startStep("Checking instance directory and backup file")
	if _, err := os.Stat(instanceDirPath); os.IsNotExist(err) {
		finishStepFatal("Instance not found")
	}

	if _, err := os.Stat(rdbPath); os.IsNotExist(err) {
		log.Fatalf("RDB file not found")
	}
	finishStep("OK")

	commandRunner := system.OSCommandRunner{
		Logger: logger,
	}

	startStep("Disabling Redis process watcher")
	if config.DedicatedInstance {
		finishStep("Skipped")
	} else {
		err = stopViaMonit(monitExecutablePath, "process-watcher")
		if err != nil {
			finishStep("ERROR")
			logger.Fatal("stop-process-watcher", err)
		}
		finishStep("OK")
	}

	processKiller := &process.ProcessKiller{}

	processController := &redis.OSProcessController{
		CommandRunner:             commandRunner,
		InstanceInformer:          &config,
		Logger:                    logger,
		ProcessChecker:            &process.ProcessChecker{},
		ProcessKiller:             processKiller,
		PingFunc:                  redis.PingServer,
		WaitUntilConnectableFunc:  availability.Check,
		RedisServerExecutablePath: config.RedisServerExecutablePath,
	}

	instance := &redis.Instance{ID: instanceID, Host: "localhost", Port: 6379}

	pidfilePath := config.InstancePidFilePath(instanceID)

	startStep("Stopping Redis")
	if config.DedicatedInstance {
		if err = unmonit(monitExecutablePath, "redis"); err == nil {
			err = processKiller.KillProvidedPID(redisPIDProvider(instanceDirPath))
		}
	} else {
		err = processController.Kill(instance)
	}

	if err != nil {
		finishStep("ERROR")
		logger.Fatal("killing-redis", err)
	}
	finishStep("OK")

	startStep("Copying backup file to instance directory")
	err = copyRdbFileIntoInstance(rdbPath, instanceDirPath)
	if err != nil {
		finishStep("ERROR")
		logger.Fatal("copy-rdb", err)
	}
	finishStep("OK")

	startStep("Starting Redis from backup file")
	err = processController.StartAndWaitUntilReadyWithConfig(
		instance,
		[]string{
			"--pidfile", pidfilePath,
			"--daemonize", "yes",
			"--dir", instanceDirPath,
			"--bind", "127.0.0.1",
		},
		time.Duration(config.StartRedisTimeoutSeconds)*time.Second,
	)
	if err != nil {
		finishStep("ERROR")
		logger.Fatal("starting-redis", err)
	}
	finishStep("OK")

	startStep("Waiting for redis to finish loading data into memory")
	client, err := client.Connect(
		client.Host(instance.Host),
		client.Port(instance.Port),
	)

	if err != nil {
		finishStep("ERROR")
		logger.Fatal("connecting-to-redis", err)
	}

	err = client.WaitUntilRedisNotLoading(config.StartRedisTimeoutSeconds * 1000)
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

	aofRewriteStatus, err := client.InfoField("aof_last_bgrewrite_status")
	if err != nil {
		logger.Fatal("getting-aof-write-status", err)
	}

	if aofRewriteStatus != "ok" {
		logger.Fatal(
			"verifying-aof-write-status",
			fmt.Errorf("Invalid AOF write status: %s", aofRewriteStatus),
		)
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
	aofPath := path.Join(instanceDirPath, "appendonly.aof")
	err = chownAof("vcap", aofPath)
	if err != nil {
		finishStep("ERROR")
		logger.Fatal("chown-aof", err)
	}
	finishStep("OK")

	startStep("Restarting Redis process watcher/redis")
	if config.DedicatedInstance {
		err = startViaMonit(monitExecutablePath, "redis")
	} else {
		err = startViaMonit(monitExecutablePath, "process-watcher")
	}

	if err != nil {
		finishStep("ERROR")
		logger.Fatal("start redis/process watcher", err)
	}
	finishStep("OK")

	fmt.Println("Restore completed successfully")
}

func redisPIDProvider(instancePath string) process.PIDProvider {
	return func() (int, error) {
		instanceConf, err := redisconf.Load(path.Join(instancePath, "redis.conf"))
		if err != nil {
			return 0, err
		}

		client, err := client.Connect(
			client.Host("localhost"),
			client.Port(instanceConf.Port()),
			client.Password(instanceConf.Password()),
			client.CmdAliases(instanceConf.CommandAliases()),
		)
		if err != nil {
			return 0, err
		}

		pidfile, err := client.GetConfig("pidfile")
		if err != nil {
			return 0, err
		}
		return process.ReadPID(pidfile)
	}
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

func startViaMonit(monitExecutablePath, processName string) error {
	_, err := monit(monitExecutablePath, []string{"start", processName})
	if err != nil {
		return err
	}

	err = waitUntilMonitStatus(monitExecutablePath, processName, "running", monitProcessRunningTimeoutMilliseconds)
	if err != nil {
		return err
	}

	return nil
}

func unmonit(monitExecutablePath, processName string) error {
	_, err := monit(monitExecutablePath, []string{"unmonitor", processName})
	return err
}

func stopViaMonit(monitExecutablePath, processName string) error {
	_, err := monit(monitExecutablePath, []string{"stop", processName})
	if err != nil {
		return err
	}

	err = waitUntilMonitStatus(monitExecutablePath, processName, "not monitored", monitProcessNotMonitoredTimeoutMilliseconds)
	if err != nil {
		return err
	}

	return nil
}

func waitUntilMonitStatus(monitExecutablePath, processName, status string, timeoutMilliseconds int) error {
	timeRemaining := timeoutMilliseconds
	for {
		output, err := monit(monitExecutablePath, []string{"summary"})
		if err != nil {
			return err
		}

		pattern := fmt.Sprintf("Process\\s+'%s'\\s+%s\\n", processName, status)
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
			return fmt.Errorf("Process %s did not reach '%s' after %d ms", processName, status, timeoutMilliseconds)
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

func restoreConfigPath() string {
	restoreConfigYamlPath := os.Getenv("RESTORE_CONFIG_PATH")
	if restoreConfigYamlPath == "" {
		return "/var/vcap/jobs/cf-redis-broker/config/restore.yml"
	}
	return restoreConfigYamlPath
}
