package redis

import (
	"errors"
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/pivotal-cf/cf-redis-broker/redis/client"
	"github.com/pivotal-golang/lager"

	"github.com/pivotal-cf/cf-redis-broker/system"
)

const redisStartTimeout time.Duration = 10 * time.Second

type ProcessChecker interface {
	Alive(pid int) bool
}

type ProcessKiller interface {
	Kill(pid int) error
}

type InstanceInformer interface {
	InstancePid(string) (int, error)
}

type OSProcessController struct {
	Logger                    lager.Logger
	InstanceInformer          InstanceInformer
	CommandRunner             system.CommandRunner
	ProcessChecker            ProcessChecker
	ProcessKiller             ProcessKiller
	PingFunc                  PingServerFunc
	WaitUntilConnectableFunc  WaitUntilConnectableFunc
	RedisServerExecutablePath string
}

func PingServer(instance *Instance) error {
	client, err := client.Connect(
		client.Host(instance.Host),
		client.Port(instance.Port),
		client.Password(instance.Password),
	)
	if err != nil {
		return err
	}

	return client.Ping()
}

type PingServerFunc func(instance *Instance) error
type WaitUntilConnectableFunc func(address *net.TCPAddr, timeout time.Duration) error

func (controller *OSProcessController) StartAndWaitUntilReady(instance *Instance, configPath, instanceDataDir, pidfilePath, logfilePath string, timeout time.Duration) error {
	instanceCommandArgs := []string{
		configPath,
		"--pidfile", pidfilePath,
		"--dir", instanceDataDir,
		"--logfile", logfilePath,
	}
	return controller.StartAndWaitUntilReadyWithConfig(instance, instanceCommandArgs, timeout)
}

func (controller *OSProcessController) StartAndWaitUntilReadyWithConfig(instance *Instance, instanceCommandArgs []string, timeout time.Duration) error {
	executable := "redis-server"
	if controller.RedisServerExecutablePath != "" {
		executable = controller.RedisServerExecutablePath
	}

	err := controller.CommandRunner.Run(executable, instanceCommandArgs...)
	if err != nil {
		return fmt.Errorf("redis failed to start: %s", err)
	}

	return controller.WaitUntilConnectableFunc(instance.Address(), timeout)
}

func (controller *OSProcessController) Kill(instance *Instance) error {
	pid, err := controller.InstanceInformer.InstancePid(instance.ID)
	if err != nil {
		return err
	}

	return controller.ProcessKiller.Kill(pid)
}

func (controller *OSProcessController) EnsureRunning(instance *Instance, configPath, instanceDataDir, pidfilePath, logfilePath string) error {
	pid, err := controller.InstanceInformer.InstancePid(instance.ID)

	if err == nil && controller.ProcessChecker.Alive(pid) {
		pingErr := controller.PingFunc(instance)
		if pingErr == nil {
			controller.Logger.Info(
				"redis instance already running",
				lager.Data{"instance": instance.ID},
			)
			return nil
		}

		controller.Logger.Error(
			"Failed to PING redis-server",
			err,
			lager.Data{"instance": instance.ID},
		)

		deleteErr := os.Remove(pidfilePath)
		if deleteErr != nil {
			controller.Logger.Error(
				"failed to delete stale pidfile",
				err,
				lager.Data{"instance": instance.ID},
			)
			return deleteErr
		}

		controller.Logger.Info(
			"removed stale pidfile",
			lager.Data{"instance": instance.ID},
		)

		return controller.StartAndWaitUntilReady(instance, configPath, instanceDataDir, pidfilePath, logfilePath, redisStartTimeout)
	}

	if err != nil {
		controller.Logger.Error(
			"redis instance has no pidfile",
			err,
			lager.Data{
				"instance": instance.ID,
			},
		)
	}

	controller.Logger.Info(
		"redis instance is not running",
		lager.Data{
			"instance": instance.ID,
		},
	)

	return controller.StartAndWaitUntilReady(instance, configPath, instanceDataDir, pidfilePath, logfilePath, redisStartTimeout)
}

func getRedisPort(name string) (int, error) {
	regex, err := regexp.Compile("\\d+$")
	if err != nil {
		return 0, err
	}

	portStr := regex.FindString(name)
	if portStr == "" {
		return 0, errors.New("failed to get port")
	}

	port, err := strconv.Atoi(string(portStr))
	if err != nil {
		return 0, err
	}

	return port, nil
}
