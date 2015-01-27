package agentconfig

import (
	"os"

	"github.com/cloudfoundry-incubator/candiedyaml"
)

type Config struct {
	DefaultConfPath     string `yaml:"default_conf_path"`
	ConfPath            string `yaml:"conf_path"`
	MonitExecutablePath string `yaml:"monit_executable_path"`
}

func Load(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	config := &Config{}
	if err := candiedyaml.NewDecoder(file).Decode(config); err != nil {
		return nil, err
	}

	return config, nil
}
