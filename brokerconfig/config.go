package brokerconfig

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/cloudfoundry-incubator/candiedyaml"
)

type Config struct {
	RedisConfiguration              ServiceConfiguration `yaml:"redis"`
	AuthConfiguration               AuthConfiguration    `yaml:"auth"`
	Host                            string               `yaml:"backend_host"`
	Port                            string               `yaml:"backend_port"`
	MonitExecutablePath             string               `yaml:"monit_executable_path"`
	RedisServerExecutablePath       string               `yaml:"redis_server_executable_path"`
	AgentPort                       string               `yaml:"agent_port"`
	ConsistencyVerificationInterval int                  `yaml:"consistency_check_interval_seconds"`
}

type AuthConfiguration struct {
	Password string `yaml:"password"`
	Username string `yaml:"username"`
}

type ServiceConfiguration struct {
	ServiceName                 string    `yaml:"service_name"`
	ServiceID                   string    `yaml:"service_id"`
	DedicatedVMPlanID           string    `yaml:"dedicated_vm_plan_id"`
	SharedVMPlanID              string    `yaml:"shared_vm_plan_id"`
	Host                        string    `yaml:"host"`
	DefaultConfigPath           string    `yaml:"redis_conf_path"`
	ProcessCheckIntervalSeconds int       `yaml:"process_check_interval"`
	StartRedisTimeoutSeconds    int       `yaml:"start_redis_timeout"`
	InstanceDataDirectory       string    `yaml:"data_directory"`
	PidfileDirectory            string    `yaml:"pidfile_directory"`
	InstanceLogDirectory        string    `yaml:"log_directory"`
	ServiceInstanceLimit        int       `yaml:"service_instance_limit"`
	Dedicated                   Dedicated `yaml:"dedicated"`
	Description                 string    `yaml:"description"`
	LongDescription             string    `yaml:"long_description"`
	ProviderDisplayName         string    `yaml:"provider_display_name"`
	DocumentationURL            string    `yaml:"documentation_url"`
	SupportURL                  string    `yaml:"support_url"`
	DisplayName                 string    `yaml:"display_name"`
	IconImage                   string    `yaml:"icon_image"`
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

	err = checkDedicatedNodesAreIPs(config.Dedicated.Nodes)
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

func checkDedicatedNodesAreIPs(dedicatedNodes []string) error {
	valid_ip_field := "(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)"
	ip_regex := fmt.Sprintf("^(%[1]s\\.){3}%[1]s$", valid_ip_field)

	for _, nodeAddress := range dedicatedNodes {
		match, _ := regexp.MatchString(ip_regex, strings.TrimSpace(nodeAddress))
		if !match {
			return errors.New("The broker only supports IP addresses for dedicated nodes")
		}
	}
	return nil
}
