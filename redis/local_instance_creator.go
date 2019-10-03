package redis

import (
	"errors"
	brokerapiresponses "github.com/pivotal-cf/brokerapi/domain/apiresponses"
	"time"

	"github.com/pborman/uuid"

	"github.com/pivotal-cf/cf-redis-broker/brokerconfig"
)

//go:generate counterfeiter -o fakes/fake_process_controller.go . ProcessController
type ProcessController interface {
	StartAndWaitUntilReady(instance *Instance, configPath, instanceDataDir, logfilePath string, timeout time.Duration) error
	Kill(instance *Instance) error
}

//go:generate counterfeiter -o fakes/fake_local_instance_repository.go . LocalInstanceRepository
type LocalInstanceRepository interface {
	FindByID(instanceID string) (*Instance, error)
	InstanceExists(instanceID string) (bool, error)
	Setup(instance *Instance) error
	Delete(instanceID string) error
	InstanceDataDir(instanceID string) string
	InstanceConfigPath(instanceID string) string
	InstanceLogFilePath(instanceID string) string
	InstancePidFilePath(instanceID string) string
	InstanceCount() (int, []error)
	Lock(instance *Instance) error
	Unlock(instance *Instance) error
}

type LocalInstanceCreator struct {
	LocalInstanceRepository
	FindFreePort       func() (int, error)
	ProcessController  ProcessController
	RedisConfiguration brokerconfig.ServiceConfiguration
}

func (localInstanceCreator *LocalInstanceCreator) Create(instanceID string) error {
	instanceCount, errs := localInstanceCreator.InstanceCount()
	if len(errs) > 0 {
		return errors.New("Failed to determine current instance count, view broker logs for details")
	}

	if instanceCount >= localInstanceCreator.RedisConfiguration.ServiceInstanceLimit {
		return brokerapiresponses.ErrInstanceLimitMet
	}

	port, err := localInstanceCreator.FindFreePort()
	if err != nil {
		return err
	}

	instance := &Instance{
		ID:       instanceID,
		Port:     port,
		Host:     localInstanceCreator.RedisConfiguration.Host,
		Password: uuid.NewRandom().String(),
	}

	err = localInstanceCreator.Setup(instance)
	if err != nil {
		return err
	}

	err = localInstanceCreator.startLocalInstance(instance)
	if err != nil {
		return err
	}

	err = localInstanceCreator.Unlock(instance)
	if err != nil {
		return err
	}

	return nil
}

func (localInstanceCreator *LocalInstanceCreator) Destroy(instanceID string) error {
	instance, err := localInstanceCreator.FindByID(instanceID)
	if err != nil {
		return err
	}

	err = localInstanceCreator.Lock(instance)
	if err != nil {
		return err
	}

	err = localInstanceCreator.ProcessController.Kill(instance)
	if err != nil {
		return err
	}

	return localInstanceCreator.Delete(instanceID)
}

func (localInstanceCreator *LocalInstanceCreator) startLocalInstance(instance *Instance) error {
	configPath := localInstanceCreator.InstanceConfigPath(instance.ID)
	instanceDataDir := localInstanceCreator.InstanceDataDir(instance.ID)
	logfilePath := localInstanceCreator.InstanceLogFilePath(instance.ID)

	timeout := time.Duration(localInstanceCreator.RedisConfiguration.StartRedisTimeoutSeconds) * time.Second
	return localInstanceCreator.ProcessController.StartAndWaitUntilReady(instance, configPath, instanceDataDir, logfilePath, timeout)
}
