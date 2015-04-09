package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/pivotal-cf/cf-redis-broker/backup"
	"github.com/pivotal-cf/cf-redis-broker/backupconfig"
	"github.com/pivotal-golang/lager"
)

func main() {
	logger := lager.NewLogger("backup")

	configPath := os.Getenv("BACKUP_CONFIG_PATH")
	if configPath == "" {
		logger.Fatal("BACKUP_CONFIG_PATH not set", nil)
	}

	config, err := backupconfig.Load(configPath)
	if err != nil {
		logger.Fatal("backup-config-load-failed", err)
	}

	if config.S3Configuration.BucketName == "" || config.S3Configuration.EndpointUrl == "" {
		logger.Info("s3 credentials not configured")
		os.Exit(0)
	}

	backupCreator := &backup.Backup{
		Config: config,
		Logger: logger,
	}
	backupErrors := map[string]error{}

	if config.DedicatedInstance {
		instanceID, err := getInstanceID(config)
		if err != nil || instanceID == "" {
			backupErrors[config.NodeIP] = err
		} else if err = backupCreator.Create(config.RedisDataDirectory, "", instanceID, "dedicated-vm"); err != nil {
			backupErrors[instanceID] = err
		}
	} else {
		backupErrors = backupSharedVMInstances(backupCreator, config.RedisDataDirectory)
	}

	if len(backupErrors) > 0 {
		logBackupErrors(backupErrors, logger)
		os.Exit(1)
	}
}

func getInstanceID(config *backupconfig.Config) (string, error) {
	url := fmt.Sprintf("http://%s/instance?host=%s", config.BrokerAddress, config.NodeIP)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(config.BrokerCredentials.Username, config.BrokerCredentials.Password)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Instance not found for %s", config.NodeIP)
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	result := struct {
		InstanceID string `json:"instance_id"`
	}{}
	err = json.Unmarshal(bytes, &result)
	if err != nil {
		return "", err
	}

	return result.InstanceID, nil
}

func backupSharedVMInstances(backupCreator *backup.Backup, instancesDir string) map[string]error {
	instanceDirs, err := ioutil.ReadDir(instancesDir)
	if err != nil {
		return map[string]error{"all-shared-vm-instances": err}
	}

	errors := map[string]error{}
	for _, instanceDir := range instanceDirs {
		basename := instanceDir.Name()
		if strings.HasPrefix(basename, ".") {
			continue
		}

		if err := backupCreator.Create(path.Join(instancesDir, basename), "db", basename, "shared-vm"); err != nil {
			errors[basename] = err
		}
	}
	return errors
}

func logBackupErrors(errors map[string]error, logger lager.Logger) {
	for instanceID, err := range errors {
		logger.Error("backup-failed", err, lager.Data{
			"instance_id": instanceID,
		})
	}
}
