package resetter

import (
	"net"
	"os"
	"time"

	"github.com/pivotal-cf/cf-redis-broker/redisconf"
	"github.com/pivotal-cf/redisutils/monit"
)

type checker interface {
	Check(address *net.TCPAddr, timeout time.Duration) error
}

//Resetter recycles a redis instance
type Resetter struct {
	defaultConfPath string
	liveConfPath    string
	portChecker     checker
	timeout         time.Duration
	Monit           monit.Monit
}

//New is the correct way to instantiate a Resetter
func New(defaultConfPath, liveConfPath string, portChecker checker) *Resetter {
	return &Resetter{
		defaultConfPath: defaultConfPath,
		liveConfPath:    liveConfPath,
		portChecker:     portChecker,
		timeout:         time.Second * 30,
		Monit:           monit.New(),
	}
}

//ResetRedis stops redis, clears the database and starts redis
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

func (resetter *Resetter) stopRedis() error {
	return resetter.Monit.StopAndWait("redis")
}

func (resetter *Resetter) startRedis() error {
	return resetter.Monit.StartAndWait("redis")
}

func (resetter *Resetter) deleteData() error {
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
