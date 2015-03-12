package fakes

import (
	"errors"
	"fmt"
	"os"

	"github.com/pivotal-cf/cf-redis-broker/brokerconfig"
	"github.com/pivotal-cf/cf-redis-broker/redis"
)

type FakeLocalRepository struct {
	FindFreePort       func() (int, error)
	DeletedInstanceIds []string
	CreatedInstances   []*redis.Instance
	LockedInstances    []*redis.Instance
	UnlockedInstances  []*redis.Instance
	Instances          []*redis.Instance
	InstanceCountErr   error
}

func (repo *FakeLocalRepository) InstanceDataDir(instanceID string) string     { return "" }
func (repo *FakeLocalRepository) InstanceConfigPath(instanceID string) string  { return "" }
func (repo *FakeLocalRepository) InstanceLogFilePath(instanceID string) string { return "" }
func (repo *FakeLocalRepository) InstancePidFilePath(instanceID string) string { return "" }

func (repo *FakeLocalRepository) Setup(instance *redis.Instance) error {
	repo.CreatedInstances = append(repo.CreatedInstances, instance)
	repo.Instances = append(repo.Instances, instance)
	return nil
}

func (repo *FakeLocalRepository) Lock(instance *redis.Instance) error {
	repo.LockedInstances = append(repo.LockedInstances, instance)
	return nil
}

func (repo *FakeLocalRepository) Unlock(instance *redis.Instance) error {
	repo.UnlockedInstances = append(repo.UnlockedInstances, instance)
	return nil
}

func (repo *FakeLocalRepository) InstanceBaseDir(instanceID string) string {
	path := fmt.Sprintf("/tmp/%s", instanceID)
	os.Mkdir(path, 0777)
	return path
}

func (FakeLocalRepository *FakeLocalRepository) Config() brokerconfig.ServiceConfiguration {
	return brokerconfig.ServiceConfiguration{
		Host:                  "127.0.0.1",
		DefaultConfigPath:     "/tmp/redis/conf",
		InstanceDataDirectory: "/tmp/redis",
	}
}

func (repo *FakeLocalRepository) Delete(instanceID string) error {
	repo.DeletedInstanceIds = append(repo.DeletedInstanceIds, instanceID)

	allInstances := []*redis.Instance{}

	for _, instance := range repo.Instances {
		if instanceID != instance.ID {
			allInstances = append(allInstances, instance)
		}
	}

	repo.Instances = allInstances

	return nil
}

func (repo *FakeLocalRepository) AllInstances() ([]*redis.Instance, error) {
	return repo.Instances, nil
}

func (repo *FakeLocalRepository) InstanceCount() (int, error) {
	if repo.InstanceCountErr != nil {
		return -1, repo.InstanceCountErr
	}

	return len(repo.Instances), nil
}

func (repo *FakeLocalRepository) FindByID(instanceID string) (*redis.Instance, error) {
	for _, instance := range repo.Instances {
		if instance.ID == instanceID {
			return instance, nil
		}
	}

	return nil, errors.New("instance not found")
}

func (repo *FakeLocalRepository) InstanceExists(instanceID string) (bool, error) {
	for _, instance := range repo.Instances {
		if instance.ID == instanceID {
			return true, nil
		}
	}

	return false, nil
}

func (repo *FakeLocalRepository) Bind(instanceID string, bindingID string) (*redis.Instance, error) {
	return repo.FindByID(instanceID)
}

func (repo *FakeLocalRepository) Unbind(instanceID string, bindingID string) error {
	return nil
}
