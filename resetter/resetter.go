package resetter

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"regexp"
	"time"

	"github.com/pivotal-cf/cf-redis-broker/credentials"
	"github.com/pivotal-cf/cf-redis-broker/redisconf"

	"code.google.com/p/go-uuid/uuid"
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

	credentials, err := credentials.Parse(resetter.liveConfPath)
	if err != nil {
		return err
	}

	address, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("127.0.0.1:%d", credentials.Port))
	if err != nil {
		return err
	}

	return resetter.portChecker.Check(address, resetter.timeout)
}

const (
	monitStart         = "start"
	monitStop          = "stop"
	monitSummary       = "summary"
	monitRunningStatus = "running"
	redisServer        = "redis"
	pgrep              = "pgrep"
)

func (resetter *Resetter) stopRedis() error {
	resetter.commandRunner.Run(exec.Command(resetter.monitExecutablePath, monitStop, redisServer))

	return resetter.loopWithTimeout("stopped", func() bool {
		cmd := exec.Command(pgrep, redisServer)
		output, _ := cmd.CombinedOutput()
		return len(output) == 0
	})
}

func (resetter *Resetter) startRedis() error {
	resetter.commandRunner.Run(exec.Command(resetter.monitExecutablePath, monitStart, redisServer))

	return resetter.loopWithTimeout("started", func() bool {
		output, _ := resetter.commandRunner.Run(exec.Command(resetter.monitExecutablePath, monitSummary))

		re := regexp.MustCompile(fmt.Sprintf(`%s'\s+(\w+)`, redisServer))
		redisServerStatus := re.FindAllStringSubmatch(string(output), -1)[0][1]

		return redisServerStatus == monitRunningStatus
	})
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

	conf.Set("requirepass", uuid.NewRandom().String())

	if err := conf.Save(resetter.liveConfPath); err != nil {
		return err
	}

	return nil
}
