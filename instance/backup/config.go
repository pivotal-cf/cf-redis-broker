package backup

import (
	"os"

	"github.com/cloudfoundry-incubator/candiedyaml"
)

type BackupConfig struct {
	S3Config               S3Configuration   `yaml:"s3"`
	SnapshotTimeoutSeconds int               `yaml:"snapshot_timeout_seconds"`
	NodeIP                 string            `yaml:"node_ip"`
	RedisConfigRoot        string            `yaml:"redis_config_root"`
	RedisConfigFilename    string            `yaml:"redis_config_filename"`
	PlanName               string            `yaml:"plan_name"`
	BrokerCredentials      BrokerCredentials `yaml:"broker_credentials"`
	BrokerAddress          string            `yaml:"broker_address"`
	LogFilepath            string            `yaml:"log_file_path"`
	AwsCLIPath             string            `yaml:"aws_cli_path"`
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

func LoadBackupConfig(path string) (*BackupConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	config := &BackupConfig{}
	if err := candiedyaml.NewDecoder(file).Decode(config); err != nil {
		return nil, err
	}

	return config, nil
}
