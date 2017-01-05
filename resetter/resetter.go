package resetter

import (
	"net"
	"os"
	"os/exec"
	"time"

	"github.com/pivotal-cf/cf-redis-broker/redisconf"
	"github.com/pivotal-cf/redisutils/monit"
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
	monit               monit.Monit
}

func New(defaultConfPath string,
	liveConfPath string,
	portChecker checker,
	commandRunner runner,
	monitExecutablePath string,
	monit monit.Monit,
) *Resetter {
	return &Resetter{
		defaultConfPath:     defaultConfPath,
		liveConfPath:        liveConfPath,
		portChecker:         portChecker,
		commandRunner:       commandRunner,
		monitExecutablePath: monitExecutablePath,
		timeout:             time.Second * 30,
		monit:               monit,
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
	return resetter.monit.StopAndWait("redis")
}

func (resetter *Resetter) startRedis() error {
	return resetter.monit.StartAndWait("redis")
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
