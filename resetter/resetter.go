package resetter

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/pivotal-cf/cf-redis-broker/redisconf"
)

type checker interface {
	Check(address *net.TCPAddr, timeout time.Duration) error
}

type runner interface {
	Run(command *exec.Cmd) ([]byte, error)
}

type Resetter struct {
	defaultConfPath     string
	liveConfPath        string
	portChecker         checker
	commandRunner       runner
	monitExecutablePath string
	timeout             time.Duration
}

func New(defaultConfPath string,
	liveConfPath string,
	portChecker checker,
	commandRunner runner,
	monitExecutablePath string) *Resetter {
	return &Resetter{
		defaultConfPath:     defaultConfPath,
		liveConfPath:        liveConfPath,
		portChecker:         portChecker,
		commandRunner:       commandRunner,
		monitExecutablePath: monitExecutablePath,
		timeout:             time.Second * 30,
	}
}

func (resetter *Resetter) ResetRedis() error {
	if err := resetter.stopRedis(); err != nil {
		return err
	}

	if err := resetter.deleteData(); err != nil {
		return err
	}

	if err := resetter.resetConfigWithNewPassword(); err != nil {
		return err
	}

	if err := resetter.startRedis(); err != nil {
		return err
	}

	conf, err := redisconf.Load(resetter.liveConfPath)
	if err != nil {
		return err
	}

	address, err := net.ResolveTCPAddr("tcp", "127.0.0.1:"+conf.Get("port"))
	if err != nil {
		return err
	}

	return resetter.portChecker.Check(address, resetter.timeout)
}

const (
	monitNotMonitoredStatus = "not monitored"
	monitRedisProcessPrefix = "Process 'redis'"
	monitRunningStatus      = "running"
	monitStart              = "start"
	monitStop               = "stop"
	monitSummary            = "summary"
	redisServer             = "redis"
)

func (resetter *Resetter) stopRedis() error {
	resetter.commandRunner.Run(exec.Command(resetter.monitExecutablePath, monitStop, redisServer))

	return resetter.loopWithTimeout("stopped", func() bool {
		redisServerStatus := resetter.redisProcessStatus()
		return redisServerStatus == monitNotMonitoredStatus
	})
}

func (resetter *Resetter) startRedis() error {
	resetter.commandRunner.Run(exec.Command(resetter.monitExecutablePath, monitStart, redisServer))

	return resetter.loopWithTimeout("started", func() bool {
		redisServerStatus := resetter.redisProcessStatus()
		return redisServerStatus == monitRunningStatus
	})
}

func (resetter *Resetter) redisProcessStatus() string {
	output, _ := resetter.commandRunner.Run(exec.Command(resetter.monitExecutablePath, monitSummary))
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, monitRedisProcessPrefix) {
			status := strings.Replace(line, monitRedisProcessPrefix, "", 1)
			return strings.TrimSpace(status)
		}
	}

	return ""
}

func (resetter *Resetter) loopWithTimeout(desiredState string, redisProcessAction func() bool) error {
	redisProcessInDesiredState := make(chan bool)

	go func(successChan chan<- bool) {
		for {
			if redisProcessAction() {
				successChan <- true
				return
			}
			time.Sleep(time.Millisecond * 100)
		}
	}(redisProcessInDesiredState)

	timer := time.NewTimer(resetter.timeout)
	defer timer.Stop()
	select {
	case <-redisProcessInDesiredState:
		break
	case <-timer.C:
		return errors.New(fmt.Sprintf("timed out waiting for redis process to be %s by monit after %d seconds", desiredState, resetter.timeout/time.Second))
	}

	return nil
}

func (_ *Resetter) deleteData() error {
	if err := os.Remove("appendonly.aof"); err != nil {
		return err
	}

	os.Remove("dump.rdb")
	return nil
}

func (resetter *Resetter) resetConfigWithNewPassword() error {
	conf, err := redisconf.Load(resetter.defaultConfPath)
	if err != nil {
		return err
	}

	err = conf.InitForDedicatedNode()
	if err != nil {
		return err
	}

	if err := conf.Save(resetter.liveConfPath); err != nil {
		return err
	}

	return nil
}
