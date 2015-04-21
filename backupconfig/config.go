package backupconfig

import (
	"os"

	"github.com/cloudfoundry-incubator/candiedyaml"
)

type Config struct {
	S3Configuration      S3Configuration   `yaml:"s3"`
	BGSaveTimeoutSeconds int               `yaml:"bg_save_timeout"`
	RedisDataDirectory   string            `yaml:"redis_data_directory"`
	NodeIP               string            `yaml:"node_ip"`
	DedicatedInstance    bool              `yaml:"dedicated_instance"`
	BrokerCredentials    BrokerCredentials `yaml:"broker_credentials"`
	BrokerHost           string            `yaml:"broker_host"`
	LogFilePath          string            `yaml:"log_file_path"`
	AwsCLIPath           string            `yaml:"aws_cli_path"`
}

type S3Configuration struct {
	EndpointUrl     string `yaml:"endpoint_url"`
	BucketName      string `yaml:"bucket_name"`
	AccessKeyId     string `yaml:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key"`
	Path            string `yaml:"path"`
}

type BrokerCredentials struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
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
