package resetter

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	redigo "github.com/gomodule/redigo/redis"
	"github.com/pivotal-cf/cf-redis-broker/redisconf"
	"github.com/pivotal-cf/redisutils/monit"
	"github.com/pivotal-cf/redisutils/redis"
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
	redis           redis.Redis
}

//New is the correct way to instantiate a Resetter
func New(defaultConfPath, liveConfPath string, portChecker checker) *Resetter {
	return &Resetter{
		defaultConfPath: defaultConfPath,
		liveConfPath:    liveConfPath,
		portChecker:     portChecker,
		timeout:         time.Second * 30,
		Monit:           monit.New(),
		redis:           redis.New(),
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
	err := resetter.killScript()
	if err != nil {
		return err
	}

	return resetter.Monit.StopAndWait("redis")
}

func (resetter *Resetter) killScript() error {
	connection, err := resetter.getAuthenticatedRedisConn()
	if err != nil {
		return err
	}
	defer connection.Close()

	_, err = connection.Do("SCRIPT", "KILL")
	if err != nil && !isNoScriptErr(err) {
		err = fmt.Errorf("failed to kill redis script: %s", err.Error())
		return err
	}

	return nil
}

func (resetter *Resetter) getAuthenticatedRedisConn() (redigo.Conn, error) {
	liveConf, err := redisconf.Load(resetter.liveConfPath)
	if err != nil {
		return nil, err
	}

	address := fmt.Sprintf("%s:%d", liveConf.Host(), liveConf.Port())

	connection, err := resetter.redis.Dial("tcp", address)
	if err != nil {
		return nil, err
	}

	_, err = connection.Do("AUTH", liveConf.Password())
	if err != nil {
		connection.Close()
		return nil, err
	}

	return connection, nil
}

func isNoScriptErr(err error) bool {
	return strings.Contains(err.Error(), "No scripts in execution right now")
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
