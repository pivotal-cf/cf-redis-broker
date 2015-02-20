package brokerconfig

import (
	"errors"
	"fmt"
	"os"

	"github.com/cloudfoundry-incubator/candiedyaml"
)

type Config struct {
	RedisConfiguration        ServiceConfiguration `yaml:"redis"`
	AuthConfiguration         AuthConfiguration    `yaml:"auth"`
	Host                      string               `yaml:"backend_host"`
	Port                      string               `yaml:"backend_port"`
	MonitExecutablePath       string               `yaml:"monit_executable_path"`
	RedisServerExecutablePath string               `yaml:"redis_server_executable_path"`
	AgentPort                 string               `yaml:"agent_port"`
}

type AuthConfiguration struct {
	Password string `yaml:"password"`
	Username string `yaml:"username"`
}

type ServiceConfiguration struct {
	Host                        string              `yaml:"host"`
	DefaultConfigPath           string              `yaml:"redis_conf_path"`
	ProcessCheckIntervalSeconds int                 `yaml:"process_check_interval"`
	StartRedisTimeoutSeconds    int                 `yaml:"start_redis_timeout"`
	InstanceDataDirectory       string              `yaml:"data_directory"`
	InstanceLogDirectory        string              `yaml:"log_directory"`
	ServiceInstanceLimit        int                 `yaml:"service_instance_limit"`
	BackupConfiguration         BackupConfiguration `yaml:"backup"`
	Dedicated                   Dedicated           `yaml:"dedicated"`
}

type BackupConfiguration struct {
	EndpointUrl          string `yaml:"endpoint_url"`
	BucketName           string `yaml:"bucket_name"`
	AccessKeyId          string `yaml:"access_key_id"`
	SecretAccessKey      string `yaml:"secret_access_key"`
	Path                 string `yaml:"path"`
	S3Region             string `yaml:"s3_region"`
	BGSaveTimeoutSeconds int    `yaml:"bg_save_timeout"`
}

type Dedicated struct {
	Nodes         []string `yaml:"nodes"`
	Port          int      `yaml:"port"`
	StatefilePath string   `yaml:"statefile_path"`
}

func (config *Config) DedicatedEnabled() bool {
	return len(config.RedisConfiguration.Dedicated.Nodes) > 0
}

func (config *Config) SharedEnabled() bool {
	return config.RedisConfiguration.ServiceInstanceLimit > 0
}

func ParseConfig(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}

	var config Config
	if err := candiedyaml.NewDecoder(file).Decode(&config); err != nil {
		return Config{}, err
	}

	return config, ValidateConfig(config.RedisConfiguration)
}

func ValidateConfig(config ServiceConfiguration) error {
	err := checkPathExists(config.DefaultConfigPath, "RedisConfig.DefaultRedisConfPath")
	if err != nil {
		return err
	}

	err = checkPathExists(config.InstanceDataDirectory, "RedisConfig.InstanceDataDirectory")
	if err != nil {
		return err
	}

	err = checkPathExists(config.InstanceLogDirectory, "RedisConfig.InstanceLogDirectory")
	if err != nil {
		return err
	}

	return nil
}

func checkPathExists(path string, description string) error {
	_, err := os.Stat(path)
	if err != nil {
		errMessage := fmt.Sprintf(
			"File '%s' (%s) not found",
			path,
			description)
		return errors.New(errMessage)
	}
	return nil
}

func (backupConfiguration BackupConfiguration) Enabled() bool {
	return backupConfiguration.BucketName != "" && backupConfiguration.EndpointUrl != ""
}
