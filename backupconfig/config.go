package backupconfig

import (
	"os"

	"github.com/cloudfoundry-incubator/candiedyaml"
)

type Config struct {
	S3Configuration      S3Configuration `yaml:"s3"`
	BGSaveTimeoutSeconds int             `yaml:"bg_save_timeout"`
	RedisDataDirectory   string          `yaml:"redis_data_directory"`
	NodeID               string          `yaml:"node_id"`
}

type S3Configuration struct {
	EndpointUrl     string `yaml:"endpoint_url"`
	BucketName      string `yaml:"bucket_name"`
	AccessKeyId     string `yaml:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key"`
	Path            string `yaml:"path"`
	Region          string `yaml:"region"`
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
