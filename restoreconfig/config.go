package restoreconfig

import (
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/cloudfoundry-incubator/candiedyaml"
)

type Config struct {
	MonitExecutablePath       string `yaml:"monit_executable_path"`
	RedisDataDirectory        string `yaml:"redis_data_directory"`
	RedisServerExecutablePath string `yaml:"redis_server_executable_path"`
	StartRedisTimeoutSeconds  int    `yaml:"start_redis_timeout_seconds"`
}

func Load(restoreConfigPath string) (Config, error) {
	file, err := os.Open(restoreConfigPath)
	if err != nil {
		return Config{}, err
	}

	var config Config
	if err := candiedyaml.NewDecoder(file).Decode(&config); err != nil {
		return Config{}, err
	}

	return config, nil
}

func (config *Config) InstancePidFilePath(instanceID string) string {
	return path.Join(config.RedisDataDirectory, instanceID, "redis-server.pid")
}

func (config *Config) InstancePid(instanceID string) (pid int, err error) {
	pidFilePath := config.InstancePidFilePath(instanceID)

	fileContent, pidFileErr := ioutil.ReadFile(pidFilePath)
	if pidFileErr != nil {
		return pid, pidFileErr
	}

	pidValue := strings.TrimSpace(string(fileContent))

	parsedPid, parseErr := strconv.ParseInt(pidValue, 10, 32)
	if parseErr != nil {
		return pid, parseErr
	}

	return int(parsedPid), err
}

func (config *Config) InstanceDataDir(instanceID string) string {
	return path.Join(config.RedisDataDirectory, instanceID, "db")
}
