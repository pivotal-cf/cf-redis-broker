package resetter

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	redigo "github.com/garyburd/redigo/redis"
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

// Several times try:
//   SCRIPT KILL
//   Atomically:
//     SET PASSWORD (memory)
//     Disconnect users
// SET PASSWORD (disk)
// FLUSHALL
// SAVE
// BGREWRITEAOF

//ResetRedis stops redis, clears the database and starts redis
func (resetter *Resetter) ResetRedis() error {
	conf, _ := redisconf.Load(resetter.liveConfPath)
	address := fmt.Sprintf("%s:%d", conf.Host(), conf.Port())
	connection, err := resetter.redis.Dial("tcp", address)

	if err != nil {
		return err
	}

	connection.Do("AUTH", conf.Password())

	return nil
}

func (resetter *Resetter) stopRedis(conf redisconf.Conf) (redisconf.Conf, error) {
	// MULTI
	connection, err := resetter.getAuthenticatedRedisConn(conf)
	if err != nil {
		return nil, err
	}
	defer connection.Close()

	conf.SetRandomPassword()

	err = resetter.killScript(connection)
	if err != nil {
		return nil, err
	}

	commandsToRun := []string{
		"CONFIG SET requirepass " + conf.Password(),
	}

	err = resetter.doAtomically(connection, commandsToRun)

	if err != nil {
		return nil, err
	}

	// SCRIPT KILL
	// RE-AUTH
	// SCRIPT FLUSH
	// CLIENT KILL skipme yes
	// CONFIG REWRITE
	// FLUSHALL
	// SAVE
	// BGREWRITEAOF
	// shutdown
	// EXEC

	return conf, resetter.Monit.StopAndWait("redis")
}

func (resetter *Resetter) doAtomically(connection redigo.Conn, commandsToRun []string) error {
	_, err := connection.Do("MULTI")
	if err != nil {
		return err
	}

	for _, command := range commandsToRun {
		_, err := connection.Do(command)
		if err != nil {
			return err
		}
	}

	return nil
}

func (resetter *Resetter) killScript(connection redigo.Conn) error {
	_, err := connection.Do("SCRIPT", "KILL")
	if err != nil && !isNoScriptErr(err) {
		err = fmt.Errorf("failed to kill redis script: %s", err.Error())
		return err
	}

	return nil
}

func (resetter *Resetter) getAuthenticatedRedisConn(conf redisconf.Conf) (redigo.Conn, error) {
	address := fmt.Sprintf("%s:%d", conf.Host(), conf.Port())

	connection, err := resetter.redis.Dial("tcp", address)
	if err != nil {
		return nil, err
	}

	_, err = connection.Do("AUTH", conf.Password())
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
