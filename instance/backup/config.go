package backup

import (
	"os"

	"github.com/cloudfoundry-incubator/candiedyaml"
)

type Config struct {
	S3Config            S3Configuration `yaml:"s3"`
	SnapshotTimeout     string          `yaml:"snapshot_timeout"`
	RedisDataRoot       string          `yaml:"redis_data_root"`
	RedisConfigFilename string          `yaml:"redis_config_filename"`
	PlanName            string          `yaml:"plan_name"`
	LogFilepath         string          `yaml:"log_file_path"`
	AwsCLIPath          string          `yaml:"aws_cli_path"`
}

type S3Configuration struct {
	EndpointUrl     string `yaml:"endpoint_url"`
	BucketName      string `yaml:"bucket_name"`
	AccessKeyId     string `yaml:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key"`
	Path            string `yaml:"path"`
}

func LoadConfig(path string) (*Config, error) {
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
