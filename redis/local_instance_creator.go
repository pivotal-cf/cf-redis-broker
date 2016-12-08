package redis

import (
	"errors"
	"time"

	"github.com/pborman/uuid"

	"github.com/pivotal-cf/brokerapi"
	"github.com/pivotal-cf/cf-redis-broker/brokerconfig"
	"github.com/pivotal-cf/cf-redis-broker/system"
)

type ProcessController interface {
	StartAndWaitUntilReady(instance *Instance, configPath, instanceDataDir, pidfilePath, logfilePath string, timeout time.Duration) error
	Kill(instance *Instance) error
}

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
	FreeTcpPort        system.FreeTcpPort
	ProcessController  ProcessController
	RedisConfiguration brokerconfig.ServiceConfiguration
}

func (localInstanceCreator *LocalInstanceCreator) Create(instanceID string) error {
	instanceCount, errs := localInstanceCreator.InstanceCount()
	if len(errs) > 0 {
		return errors.New("Failed to determine current instance count, view broker logs for details")
	}

	if instanceCount >= localInstanceCreator.RedisConfiguration.ServiceInstanceLimit {
		return brokerapi.ErrInstanceLimitMet
	}
	SharedMaxPort := localInstanceCreator.RedisConfiguration.SharedMaxPort
	SharedMinPort := localInstanceCreator.RedisConfiguration.SharedMinPort
	port, err := localInstanceCreator.FreeTcpPort.FindFreePortInRange(SharedMinPort, SharedMaxPort)
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
	pidfilePath := localInstanceCreator.InstancePidFilePath(instance.ID)

	timeout := time.Duration(localInstanceCreator.RedisConfiguration.StartRedisTimeoutSeconds) * time.Second
	return localInstanceCreator.ProcessController.StartAndWaitUntilReady(instance, configPath, instanceDataDir, pidfilePath, logfilePath, timeout)
}
