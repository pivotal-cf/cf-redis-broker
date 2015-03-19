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

		if err := moveDataFor("port", redisPortFilePath, redisConfFilePath); err != nil {
			return err
		}

		if err := moveDataFor("requirepass", redisPasswordFilePath, redisConfFilePath); err != nil {
			return err
		}

	}
	return nil
}

func moveDataFor(propertyName, fileName, redisConfFilePath string) error {
	var fileContents []byte
	var redisConf redisconf.Conf
	var err error

	if redisConf, err = redisconf.Load(redisConfFilePath); err != nil {
		return err
	}
	if redisConf.Get(propertyName) != "" {
		os.Remove(fileName)
		return nil
	}

	if fileContents, err = ioutil.ReadFile(fileName); err != nil {
		return err
	}

	redisConf.Set(propertyName, string(fileContents))
	if err = redisConf.Save(redisConfFilePath); err != nil {
		return err
	}
	os.Remove(fileName)
	return nil
}
