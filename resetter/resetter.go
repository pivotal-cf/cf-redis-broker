package resetter

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
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
		timeout:             time.Second * 10,
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

func (resetter *Resetter) stopRedis() error {
	resetter.commandRunner.Run(exec.Command(resetter.monitExecutablePath, "stop", "redis"))
	redisProcessDead := make(chan bool)
	go func(c chan<- bool) {
		for {
			cmd := exec.Command("pgrep", "redis-server")
			output, _ := cmd.CombinedOutput()
			if len(output) == 0 {
				c <- true
				return
			}
			time.Sleep(time.Millisecond * 100)
		}
	}(redisProcessDead)

	timer := time.NewTimer(resetter.timeout)
	defer timer.Stop()
	select {
	case <-redisProcessDead:
		break
	case <-timer.C:
		return errors.New("timed out waiting for redis process to die after 10 seconds")
	}

	return nil
}

func (resetter *Resetter) startRedis() error {
	redisStarted := make(chan bool)
	go func(c chan<- bool) {
		for {
			_, err := resetter.commandRunner.Run(exec.Command(resetter.monitExecutablePath, "start", "redis"))
			if err == nil {
				c <- true
				return
			}
			time.Sleep(time.Millisecond * 100)
		}
	}(redisStarted)

	timer := time.NewTimer(resetter.timeout)
	defer timer.Stop()
	select {
	case <-redisStarted:
		break
	case <-timer.C:
		return errors.New("timed out waiting for redis process to be started by monit after 10 seconds")
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
