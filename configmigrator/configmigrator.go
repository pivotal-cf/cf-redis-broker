package configmigrator

import (
	"io/ioutil"
	"os"
	"path"

	"github.com/pivotal-cf/cf-redis-broker/redisconf"
)

type ConfigMigrator struct {
	RedisDataDir string
}

func (migrator *ConfigMigrator) Migrate() error {
	instanceDirs, _ := ioutil.ReadDir(migrator.RedisDataDir)
	redisInstanceDir := path.Join(migrator.RedisDataDir, instanceDirs[0].Name())
	redisPortFilePath := path.Join(redisInstanceDir, "redis.port")
	redisConfFilePath := path.Join(redisInstanceDir, "redis.conf")

	redisConf, _ := redisconf.Load(redisConfFilePath)

	portBytes, _ := ioutil.ReadFile(redisPortFilePath)
	redisConf.Set("port", string(portBytes))

	redisConf.Save(redisConfFilePath)

	return os.Remove(redisPortFilePath)
}
