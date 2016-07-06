package redis

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pivotal-cf/cf-redis-broker/broker"
	"github.com/pivotal-cf/cf-redis-broker/brokerconfig"
	"github.com/pivotal-cf/cf-redis-broker/redisconf"
	"github.com/pivotal-golang/lager"
)

type LocalRepository struct {
	RedisConf brokerconfig.ServiceConfiguration
	Logger    lager.Logger
}

func NewLocalRepository(redisConf brokerconfig.ServiceConfiguration, logger lager.Logger) *LocalRepository {
	return &LocalRepository{
		RedisConf: redisConf,
		Logger:    logger,
	}
}

func (repo *LocalRepository) FindByID(instanceID string) (*Instance, error) {
	conf, err := redisconf.Load(repo.InstanceConfigPath(instanceID))
	if err != nil {
		return nil, err
	}

	port, err := strconv.Atoi(conf.Get("port"))
	if err != nil {
		return nil, err
	}

	instance := &Instance{
		ID:       instanceID,
		Password: conf.Get("requirepass"),
		Port:     port,
		Host:     repo.RedisConf.Host,
	}

	return instance, nil
}

func (repo *LocalRepository) InstanceExists(instanceID string) (bool, error) {
	if _, err := os.Stat(repo.InstanceBaseDir(instanceID)); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

// Eventually: make lock the first thing to be called
// EnsureDirectoriesExist -> EnsureLogDirectoryExists

func (repo *LocalRepository) Setup(instance *Instance) error {
	err := repo.EnsureDirectoriesExist(instance)
	if err != nil {
		repo.Logger.Error("ensure-dirs-exist", err, lager.Data{
			"instance_id": instance.ID,
		})
		return err
	}

	err = repo.Lock(instance)
	if err != nil {
		repo.Logger.Error("lock-shared-instance", err, lager.Data{
			"instance_id": instance.ID,
		})
		return err
	}

	err = repo.WriteConfigFile(instance)
	if err != nil {
		repo.Logger.Error("write-config-file", err, lager.Data{
			"instance_id": instance.ID,
		})
		return err
	}

	return nil
}

func (repo *LocalRepository) Lock(instance *Instance) error {
	lockFilePath := repo.lockFilePath(instance)
	lockFile, err := os.Create(lockFilePath)
	if err != nil {
		return err
	}
	lockFile.Close()

	return nil
}

func (repo *LocalRepository) Unlock(instance *Instance) error {
	lockFilePath := repo.lockFilePath(instance)
	err := os.Remove(lockFilePath)
	if err != nil {
		return err
	}

	return nil
}

func (repo *LocalRepository) lockFilePath(instance *Instance) string {
	return filepath.Join(repo.InstanceBaseDir(instance.ID), "lock")
}

func (repo *LocalRepository) allInstances(verbose bool) ([]*Instance, []error) {
	if verbose {
		repo.Logger.Info("all-instances", lager.Data{
			"message": fmt.Sprintf("Starting shared instance lookup in data directory: %s", repo.RedisConf.InstanceDataDirectory),
		})
	}

	instances := []*Instance{}

	instanceDirs, err := ioutil.ReadDir(repo.RedisConf.InstanceDataDirectory)
	if err != nil {
		repo.Logger.Error("all-instances", err, lager.Data{
			"message":        "Error finding shared instances",
			"data-directory": repo.RedisConf.InstanceDataDirectory,
		})
		return instances, []error{err}
	}

	var pluralisedInstance string
	if len(instanceDirs) == 1 {
		pluralisedInstance = "instance"
	} else {
		pluralisedInstance = "instances"
	}

	if verbose {
		repo.Logger.Info("all-instances", lager.Data{
			"message": fmt.Sprintf("%d shared Redis %s found", len(instanceDirs), pluralisedInstance),
		})
	}

	errs := []error{}

	for _, instanceDir := range instanceDirs {

		instance, err := repo.FindByID(instanceDir.Name())

		if err != nil {
			repo.Logger.Error("all-instances", err, lager.Data{
				"message": fmt.Sprintf("Error getting instance details for instance ID: %s", instanceDir.Name()),
			})

			errs = append(errs, err)
			continue
		}

		if verbose {
			repo.Logger.Info("all-instances", lager.Data{
				"message": fmt.Sprintf("Found shared instance: %s", instance.ID),
			})
		}

		instances = append(instances, instance)
	}

	return instances, errs
}

func (repo *LocalRepository) AllInstancesVerbose() ([]*Instance, []error) {
	return repo.allInstances(true)
}

func (repo *LocalRepository) AllInstances() ([]*Instance, []error) {
	return repo.allInstances(false)
}

func (repo *LocalRepository) InstanceCount() (int, []error) {
	instances, errs := repo.AllInstances()
	return len(instances), errs
}

func (repo *LocalRepository) Bind(instanceID string, bindingID string) (broker.InstanceCredentials, error) {
	instance, err := repo.FindByID(instanceID)
	if err != nil {
		return broker.InstanceCredentials{}, err
	}
	return broker.InstanceCredentials{
		Host:     instance.Host,
		Port:     instance.Port,
		Password: instance.Password,
	}, nil
}

func (repo *LocalRepository) Unbind(instanceID string, bindingID string) error {
	return nil
}

func (repo *LocalRepository) Delete(instanceID string) error {
	err := os.RemoveAll(repo.InstanceBaseDir(instanceID))
	if err != nil {
		return err
	}

	err = os.RemoveAll(repo.InstanceLogDir(instanceID))
	if err != nil {
		return err
	}

	err = os.Remove(repo.InstancePidFilePath(instanceID))
	if err != nil {
		return err
	}

	return nil
}

func (repo *LocalRepository) EnsureDirectoriesExist(instance *Instance) error {
	err := os.MkdirAll(repo.InstanceDataDir(instance.ID), 0755)
	if err != nil {
		return err
	}

	err = os.MkdirAll(repo.RedisConf.PidfileDirectory, 0755)
	if err != nil {
		return err
	}

	err = os.MkdirAll(repo.InstanceLogDir(instance.ID), 0755)
	if err != nil {
		return err
	}

	return nil
}

func (repo *LocalRepository) WriteConfigFile(instance *Instance) error {
	return redisconf.CopyWithInstanceAdditions(
		repo.RedisConf.DefaultConfigPath,
		repo.InstanceConfigPath(instance.ID),
		instance.ID,
		strconv.Itoa(instance.Port),
		instance.Password,
	)
}

func (repo *LocalRepository) InstanceBaseDir(instanceID string) string {
	return path.Join(repo.RedisConf.InstanceDataDirectory, instanceID)
}

func (repo *LocalRepository) InstanceDataDir(instanceID string) string {
	InstanceBaseDir := repo.InstanceBaseDir(instanceID)
	return path.Join(InstanceBaseDir, "db")
}

func (repo *LocalRepository) InstanceLogDir(instanceID string) string {
	return path.Join(repo.RedisConf.InstanceLogDirectory, instanceID)
}

func (repo *LocalRepository) InstanceLogFilePath(instanceID string) string {
	return path.Join(repo.InstanceLogDir(instanceID), "redis-server.log")
}

func (repo *LocalRepository) InstanceConfigPath(instanceID string) string {
	return path.Join(repo.InstanceBaseDir(instanceID), "redis.conf")
}

func (repo *LocalRepository) InstancePidFilePath(instanceID string) string {
	return path.Join(repo.RedisConf.PidfileDirectory, instanceID+".pid")
}

func (repo *LocalRepository) InstancePid(instanceID string) (pid int, err error) {
	pidFilePath := repo.InstancePidFilePath(instanceID)

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
