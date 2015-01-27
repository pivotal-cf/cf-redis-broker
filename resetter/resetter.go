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

type portChecker interface {
	Check(address *net.TCPAddr, timeout time.Duration) error
}

type Shell interface {
	Run(command *exec.Cmd) ([]byte, error)
}

type Resetter struct {
	defaultConfPath string
	confPath        string
	portChecker     portChecker
	shell           Shell
	monitExecutable string
}

func New(defaultConfPath string, confPath string, portChecker portChecker, shell Shell, monitExecutable string) *Resetter {
	return &Resetter{
		defaultConfPath: defaultConfPath,
		confPath:        confPath,
		portChecker:     portChecker,
		shell:           shell,
		monitExecutable: monitExecutable,
	}
}

func (Resetter *Resetter) DeleteAllData() error {
	Resetter.shell.Run(exec.Command(Resetter.monitExecutable, "stop", "redis"))
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

	select {
	case <-redisProcessDead:
	case <-time.After(time.Second * 10):
		return errors.New("timed out waiting for redis process to die after 10 seconds")
	}

	err := os.Remove("appendonly.aof")
	if err != nil {
		return err
	}
	os.Remove("dump.rdb")
	err = Resetter.resetConf()
	if err != nil {
		return err
	}
	redisStarted := make(chan bool)
	go func(c chan<- bool) {
		for {
			_, err := Resetter.shell.Run(exec.Command(Resetter.monitExecutable, "start", "redis"))
			if err == nil {
				c <- true
				return
			}
			time.Sleep(time.Millisecond * 100)
		}
	}(redisStarted)
	select {
	case <-redisStarted:
	case <-time.After(time.Second * 10):
		return errors.New("timed out waiting for redis process to be started by monit after 10 seconds")
	}
	credentials, err := credentials.Parse(Resetter.confPath)
	if err != nil {
		return err
	}
	address, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("127.0.0.1:%d", credentials.Port))
	if err != nil {
		return err
	}
	return Resetter.portChecker.Check(address, time.Second*10)
}

func (Resetter *Resetter) resetConf() error {
	conf, err := redisconf.Load(Resetter.defaultConfPath)
	if err != nil {
		return err
	}

	conf.Set("requirepass", uuid.NewRandom().String())
	err = conf.Save(Resetter.confPath)
	if err != nil {
		return err
	}

	return nil
}
