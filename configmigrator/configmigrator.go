package configmigrator

import (
	"io/ioutil"
	"os"
	"path"

	"github.com/pivotal-cf/cf-redis-broker/redisconf"
)

const (
	REDIS_PORT_FILENAME     = "redis-server.port"
	REDIS_PASSWORD_FILENAME = "redis-server.password"
)

type ConfigMigrator struct {
	RedisDataDir string
}

func (migrator *ConfigMigrator) Migrate() error {
	instanceDirs, _ := ioutil.ReadDir(migrator.RedisDataDir)

	for _, instanceDir := range instanceDirs {
		redisInstanceDir := path.Join(migrator.RedisDataDir, instanceDir.Name())
		redisPortFilePath := path.Join(redisInstanceDir, REDIS_PORT_FILENAME)
		redisPasswordFilePath := path.Join(redisInstanceDir, REDIS_PASSWORD_FILENAME)
		redisConfFilePath := path.Join(redisInstanceDir, "redis.conf")

		var err error
		var redisConf redisconf.Conf
		var portBytes []byte
		var passwordBytes []byte

		if redisConf, err = redisconf.Load(redisConfFilePath); err != nil {
			return err
		}

		if portBytes, err = ioutil.ReadFile(redisPortFilePath); err != nil {
			return err
		}

		if passwordBytes, err = ioutil.ReadFile(redisPasswordFilePath); err != nil {
			return err
		}

		redisConf.Set("port", string(portBytes))
		redisConf.Set("requirepass", string(passwordBytes))
		if err = redisConf.Save(redisConfFilePath); err != nil {
			return err
		}

		os.Remove(redisPortFilePath)
		os.Remove(redisPasswordFilePath)
	}
	return nil
}
