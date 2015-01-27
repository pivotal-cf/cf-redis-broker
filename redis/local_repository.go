package redis

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pivotal-cf/cf-redis-broker/broker"
	"github.com/pivotal-cf/cf-redis-broker/brokerconfig"
	"github.com/pivotal-cf/cf-redis-broker/redis/config"
)

type LocalRepository struct {
	RedisConf brokerconfig.ServiceConfiguration
}

type InstanceNotFoundErr struct{}

func (err InstanceNotFoundErr) Error() string {
	return "Instance not found"
}

func (repo *LocalRepository) FindByID(instanceID string) (*Instance, error) {
	instanceBaseDir := repo.InstanceBaseDir(instanceID)

	_, pathErr := os.Stat(instanceBaseDir)
	if pathErr != nil {
		return nil, InstanceNotFoundErr{}
	}

	passwordBytes, passwordReadErr := ioutil.ReadFile(path.Join(instanceBaseDir, "redis-server.password"))
	if passwordReadErr != nil {
		return nil, passwordReadErr
	}

	portBytes, portReadErr := ioutil.ReadFile(path.Join(instanceBaseDir, "redis-server.port"))
	if portReadErr != nil {
		return nil, portReadErr
	}

	portString := string(portBytes)
	port, err := strconv.Atoi(strings.TrimSpace(portString))
	if err != nil {
		return nil, err
	}

	instance := &Instance{
		ID:       instanceID,
		Password: string(passwordBytes),
		Port:     port,
		Host:     repo.RedisConf.Host,
	}

	return instance, nil
}

func (repo *LocalRepository) InstanceExists(instanceID string) (bool, error) {
	_, err := repo.FindByID(instanceID)
	if _, ok := err.(InstanceNotFoundErr); ok {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

// Eventually: make lock the first thing to be called
// EnsureDirectoriesExist -> EnsureLogDirectoryExists

func (repo *LocalRepository) Setup(instance *Instance) error {
	repo.EnsureDirectoriesExist(instance)
	repo.Lock(instance)
	repo.WriteConfigFile(instance)
	repo.WriteBindingData(instance)

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

func (repo *LocalRepository) AllInstances() ([]*Instance, error) {
	instances := []*Instance{}

	instanceDirs, err := ioutil.ReadDir(repo.RedisConf.InstanceDataDirectory)
	if err != nil {
		return instances, err
	}

	for _, instanceDir := range instanceDirs {

		instance, err := repo.FindByID(instanceDir.Name())

		if err != nil {
			return instances, err
		}

		instances = append(instances, instance)
	}

	return instances, nil
}

func (repo *LocalRepository) InstanceCount() (int, error) {
	instances, err := repo.AllInstances()
	return len(instances), err
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

	return nil
}

func (repo *LocalRepository) EnsureDirectoriesExist(instance *Instance) error {
	err := os.MkdirAll(repo.InstanceDataDir(instance.ID), 0755)
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
	defaultConfigPath := repo.RedisConf.DefaultConfigPath
	InstanceConfigPath := repo.InstanceConfigPath(instance.ID)

	err := config.SaveRedisConfAdditions(defaultConfigPath, InstanceConfigPath, instance.ID)
	if err != nil {
		return err
	}

	return nil
}

func (repo *LocalRepository) WriteBindingData(instance *Instance) error {

	port := strconv.FormatInt(int64(instance.Port), 10)

	baseDir := repo.InstanceBaseDir(instance.ID)

	ioutil.WriteFile(path.Join(baseDir, "redis-server.password"), []byte(instance.Password), 0644)
	ioutil.WriteFile(path.Join(baseDir, "redis-server.port"), []byte(port), 0644)

	return nil
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
	return path.Join(repo.InstanceBaseDir(instanceID), "redis-server.pid")
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
