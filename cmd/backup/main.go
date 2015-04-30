package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/pivotal-cf/cf-redis-broker/backup"
	"github.com/pivotal-cf/cf-redis-broker/backupconfig"
	"github.com/pivotal-golang/lager"
)

var logger lager.Logger

func main() {
	config := loadConfig()
	logger = initializeLogger(config)

	logger.Info("backup-main", lager.Data{
		"event": "starting",
	})

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

			logger.Info("backup-main", lager.Data{
				"event": "backup_dedicated_node",
				"error": err.Error(),
			})

		} else if err = backupCreator.Create(config.RedisDataDirectory, "", instanceID, "dedicated-vm"); err != nil {
			logger.Error("backup-main", err, lager.Data{
				"event": "backup_creator",
			})

			backupErrors[instanceID] = err
		}
	} else {
		backupErrors = backupSharedVMInstances(backupCreator, config.RedisDataDirectory)
	}

	if len(backupErrors) > 0 {
		logBackupErrors(backupErrors, logger)

		logger.Info("backup-main", lager.Data{
			"event":     "Exiting",
			"exit_code": 1,
		})
		os.Exit(1)
	}

	logger.Info("backup-main", lager.Data{
		"event":     "Exiting",
		"exit_code": 0,
	})
}

func getInstanceID(config *backupconfig.Config) (string, error) {
	url := fmt.Sprintf("http://%s/instance?host=%s", config.BrokerHost, config.NodeIP)
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
	logger.Info("backup-main", lager.Data{
		"event": "backup_shared_vm_instances",
	})

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

func loadConfig() *backupconfig.Config {
	configPath := os.Getenv("BACKUP_CONFIG_PATH")
	if configPath == "" {
		log.Fatal("BACKUP_CONFIG_PATH not set", nil)
	}

	config, err := backupconfig.Load(configPath)
	if err != nil {
		log.Fatal("backup-config-load-failed", err)
	}
	return config
}

func initializeLogger(config *backupconfig.Config) lager.Logger {
	logger := lager.NewLogger("backup")
	logFile, err := os.OpenFile(config.LogFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0660)
	if err != nil {
		log.Fatal("unable to open log file")
	}
	logger.RegisterSink(lager.NewWriterSink(logFile, lager.INFO))
	return logger
}
