package redis

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	"github.com/pivotal-cf/brokerapi"
	"github.com/pivotal-cf/cf-redis-broker/broker"
	"github.com/pivotal-cf/cf-redis-broker/brokerconfig"
	"github.com/pivotal-golang/lager"
)

type RemoteRepository struct {
	availableInstances []*Instance
	allocatedInstances []*Instance
	instanceLimit      int
	instanceBindings   map[string][]string
	agentClient        AgentClient
	statefilePath      string
	agentPort          string
	sync.RWMutex
	logger lager.Logger
}

type AgentClient interface {
	Reset(hostIP string) error
	Credentials(hostIP string) (Credentials, error)
}

func NewRemoteRepository(agentClient AgentClient, config brokerconfig.Config, logger lager.Logger) (*RemoteRepository, error) {
	repo := RemoteRepository{
		instanceLimit:    len(config.RedisConfiguration.Dedicated.Nodes),
		instanceBindings: make(map[string][]string),
		statefilePath:    config.RedisConfiguration.Dedicated.StatefilePath,
		agentClient:      agentClient,
		agentPort:        config.AgentPort,
		logger:           logger,
	}

	err := repo.loadStateFromFile()
	if err != nil {
		return nil, err
	}

	for _, ip := range config.RedisConfiguration.Dedicated.Nodes {
		available := true
		for _, allocatedInstance := range repo.allocatedInstances {
			if ip == allocatedInstance.Host {
				available = false
				break
			}
		}
		if available {
			instance := Instance{
				Host: ip,
			}
			repo.availableInstances = append(repo.availableInstances, &instance)
		}
	}

	err = repo.PersistStatefile()
	if err != nil {
		return nil, err
	}

	return &repo, nil
}

func (repo *RemoteRepository) FindByID(instanceID string) (*Instance, error) {
	for _, instance := range repo.allocatedInstances {
		if instance.ID == instanceID {
			return instance, nil
		}
	}
	return nil, brokerapi.ErrInstanceDoesNotExist
}

func (repo *RemoteRepository) InstanceExists(instanceID string) (bool, error) {
	_, err := repo.FindByID(instanceID)
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (repo *RemoteRepository) Destroy(instanceID string) error {
	repo.Lock()
	defer repo.Unlock()

	instance, err := repo.FindByID(instanceID)
	if err != nil {
		return err
	}

	instance.ID = instanceID

	instanceURL := "https://" + instance.Host + ":" + repo.agentPort
	err = repo.agentClient.Reset(instanceURL)
	if err != nil {
		return err
	}

	repo.deallocateInstance(instance)

	err = repo.PersistStatefile()
	if err != nil {
		repo.allocateInstance(instanceID)
		return err
	}

	return nil
}

func (repo *RemoteRepository) AllInstances() ([]*Instance, error) {
	return repo.allocatedInstances, nil
}

func (repo *RemoteRepository) InstanceCount() (int, error) {
	return len(repo.allocatedInstances), nil
}

func (repo *RemoteRepository) Create(instanceID string) error {
	repo.Lock()
	defer repo.Unlock()

	if len(repo.availableInstances) <= 0 {
		return brokerapi.ErrInstanceLimitMet
	}

	existingInstance, _ := repo.FindByID(instanceID)
	if existingInstance != nil {
		return brokerapi.ErrInstanceAlreadyExists
	}

	instance := repo.allocateInstance(instanceID)

	err := repo.PersistStatefile()
	if err != nil {
		repo.deallocateInstance(instance)
		return err
	}

	return nil
}

func (repo *RemoteRepository) Bind(instanceID string, bindingID string) (broker.InstanceCredentials, error) {
	repo.Lock()
	defer repo.Unlock()

	instance, err := repo.FindByID(instanceID)
	if err != nil {
		return broker.InstanceCredentials{}, err
	}

	bindings, _ := repo.instanceBindings[instanceID]
	for _, binding := range bindings {
		if binding == bindingID {
			return broker.InstanceCredentials{}, brokerapi.ErrBindingAlreadyExists
		}
	}

	instanceURL := "https://" + instance.Host + ":" + repo.agentPort
	credentials, err := repo.agentClient.Credentials(instanceURL)
	if err != nil {
		return broker.InstanceCredentials{}, err
	}

	instance.Port = credentials.Port
	instance.Password = credentials.Password

	repo.instanceBindings[instanceID] = append(repo.instanceBindings[instanceID], bindingID)

	err = repo.PersistStatefile()
	if err != nil {
		repo.removeBinding(instanceID, bindingID)
		return broker.InstanceCredentials{}, err
	}

	return broker.InstanceCredentials{
		Host:     instance.Host,
		Port:     instance.Port,
		Password: instance.Password,
	}, nil
}

func (repo *RemoteRepository) Unbind(instanceID string, bindingID string) error {
	repo.Lock()
	defer repo.Unlock()

	if _, err := repo.FindByID(instanceID); err != nil {
		return err
	}

	bindings, _ := repo.instanceBindings[instanceID]

	for _, binding := range bindings {

		if binding == bindingID {
			err := repo.removeBinding(instanceID, bindingID)
			if err != nil {
				return err
			}

			err = repo.PersistStatefile()
			if err != nil {
				repo.instanceBindings[instanceID] = append(repo.instanceBindings[instanceID], bindingID)
				return err
			}

			return nil
		}

	}

	return brokerapi.ErrBindingDoesNotExist
}

func (repo *RemoteRepository) InstanceLimit() int {
	return repo.instanceLimit
}

func (repo *RemoteRepository) AvailableInstances() []*Instance {
	return repo.availableInstances
}

func (repo *RemoteRepository) BindingsForInstance(instanceID string) ([]string, error) {
	bindings, ok := repo.instanceBindings[instanceID]
	if !ok {
		return nil, brokerapi.ErrInstanceDoesNotExist
	}

	return bindings, nil
}

type Statefile struct {
	AvailableInstances []*Instance         `json:"available_instances"`
	AllocatedInstances []*Instance         `json:"allocated_instances"`
	InstanceBindings   map[string][]string `json:"instance_bindings"`
}

func newStatefile() Statefile {
	return Statefile{InstanceBindings: make(map[string][]string)}
}

func (repo *RemoteRepository) PersistStatefile() error {
	statefileContents := Statefile{
		AvailableInstances: repo.availableInstances,
		AllocatedInstances: repo.allocatedInstances,
		InstanceBindings:   repo.instanceBindings,
	}

	stateBytes, err := json.Marshal(&statefileContents)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(repo.statefilePath, stateBytes, 0644)
}

func (repo *RemoteRepository) IDForHost(host string) string {
	for _, instance := range repo.allocatedInstances {
		if instance.Host == host {
			return instance.ID
		}
	}
	return ""
}

func (repo *RemoteRepository) StateFromFile() (Statefile, error) {
	repo.logger.Info(fmt.Sprintf("Starting dedicated instance lookup in statefile: %s", repo.statefilePath))

	statefileContents := newStatefile()

	if _, err := os.Stat(repo.statefilePath); os.IsNotExist(err) {
		repo.logger.Info(fmt.Sprintf(
			"statefile %s not found, generating instead",
			repo.statefilePath,
		))
		repo.logger.Info("all-instances", lager.Data{
			"message": "0 dedicated Redis instances found",
		})
		return statefileContents, nil
	}

	stateBytes, err := ioutil.ReadFile(repo.statefilePath)
	if err != nil {
		repo.logger.Error(
			"failed to read statefile",
			err, lager.Data{"statefilePath": repo.statefilePath},
		)
		return statefileContents, err
	}

	err = json.Unmarshal(stateBytes, &statefileContents)
	if err != nil {
		repo.logger.Error(
			"failed to read statefile due to invalid JSON",
			err,
			lager.Data{
				"statefilePath":     repo.statefilePath,
				"stateFileContents": string(stateBytes),
			},
		)
		return statefileContents, err
	}

	var pluralisedInstance string

	if len(statefileContents.AllocatedInstances) == 1 {
		pluralisedInstance = "instance"
	} else {
		pluralisedInstance = "instances"
	}

	repo.logger.Info("all-instances", lager.Data{
		"message": fmt.Sprintf("%d dedicated Redis %s found", len(statefileContents.AllocatedInstances), pluralisedInstance),
	})

	for _, instance := range statefileContents.AllocatedInstances {
		repo.logger.Info("all-instances", lager.Data{
			"message": fmt.Sprintf("Found dedicated instance: %s", instance.ID),
		})
	}

	return statefileContents, nil
}

func (repo *RemoteRepository) loadStateFromFile() error {
	statefileContents, err := repo.StateFromFile()
	if err != nil {
		return err
	}

	repo.allocatedInstances = statefileContents.AllocatedInstances
	repo.instanceBindings = statefileContents.InstanceBindings

	return nil
}

func (repo *RemoteRepository) removeBinding(instanceID, bindingID string) error {
	var newInstanceBindings []string

	_, err := repo.FindByID(instanceID)
	if err != nil {
		return err
	}

	bindings, _ := repo.instanceBindings[instanceID]
	found := false
	for _, binding := range bindings {
		if binding != bindingID {
			newInstanceBindings = append(newInstanceBindings, binding)
		} else {
			found = true
		}
	}

	if !found {
		return errors.New("binding not found")
	}

	repo.instanceBindings[instanceID] = newInstanceBindings

	return nil
}

func (repo *RemoteRepository) allocateInstance(instanceID string) *Instance {

	instance := repo.availableInstances[0]
	repo.availableInstances = repo.availableInstances[1:]

	instance.ID = instanceID
	repo.allocatedInstances = append(repo.allocatedInstances, instance)

	repo.instanceBindings[instanceID] = []string{}

	return instance
}

func (repo *RemoteRepository) deallocateInstance(instance *Instance) {
	nowAllocatedInstances := []*Instance{}
	for _, previouslyAllocatedInstance := range repo.allocatedInstances {
		if previouslyAllocatedInstance.Host != instance.Host {
			nowAllocatedInstances = append(nowAllocatedInstances, previouslyAllocatedInstance)
		}
	}

	repo.allocatedInstances = nowAllocatedInstances

	repo.availableInstances = append([]*Instance{instance}, repo.availableInstances...)

	delete(repo.instanceBindings, instance.ID)
}
