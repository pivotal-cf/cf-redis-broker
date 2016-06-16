package redis

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"
	"time"

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
	WaitUntilConnectableFunc  WaitUntilConnectableFunc
	RedisServerExecutablePath string
}

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
		output, psErr := psPID(controller, instance.ID)
		if psErr != nil {
			return psErr
		}

		if bytes.Contains(output, []byte("redis-server")) {
			port, portErr := getPortFromPS(controller, output)
			if portErr != nil {
				return portErr
			}

			if port == instance.Port {
				controller.Logger.Debug(
					"redis instance already running",
					lager.Data{"instance": instance.ID},
				)
				return nil
			}
		}

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

func psPID(controller *OSProcessController, pid string) ([]byte, error) {
	output, err := controller.CommandRunner.CombinedOutput(
		"ps",
		"aux",
		"|",
		"grep",
		fmt.Sprintf(`"vcap\\s*%s"`, pid),
	)

	if err != nil {
		controller.Logger.Error(
			"ps failed for pid",
			err,
			lager.Data{"pid": pid},
		)
		return []byte{}, err
	}

	return output, err
}

func getPortFromPS(controller *OSProcessController, output []byte) (int, error) {
	regex, err := regexp.Compile("\\d+$")
	if err != nil {
		return 0, err
	}

	portStr := regex.Find(output)
	if portStr == nil {
		return 0, errors.New("failed to get port")
	}

	port, err := strconv.Atoi(string(portStr))
	if err != nil {
		return 0, err
	}

	return port, nil
}
