package configmigrator

import (
	"io/ioutil"
	"os"
	"path"

	"github.com/pivotal-cf/cf-redis-broker/redisconf"
)

const (
	REDIS_PORT_FILENAME = "redis-server.port"
)

type ConfigMigrator struct {
	RedisDataDir string
}

func (migrator *ConfigMigrator) Migrate() error {
	instanceDirs, _ := ioutil.ReadDir(migrator.RedisDataDir)
	redisInstanceDir := path.Join(migrator.RedisDataDir, instanceDirs[0].Name())
	redisPortFilePath := path.Join(redisInstanceDir, REDIS_PORT_FILENAME)
	redisConfFilePath := path.Join(redisInstanceDir, "redis.conf")

	redisConf, err := redisconf.Load(redisConfFilePath)
	if err != nil {
		return err
	}

	portBytes, err := ioutil.ReadFile(redisPortFilePath)
	if err == nil {
		redisConf.Set("port", string(portBytes))
		redisConf.Save(redisConfFilePath)
		return os.Remove(redisPortFilePath)
	}
	return nil
}
