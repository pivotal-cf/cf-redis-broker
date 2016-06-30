package broker

import (
	"errors"
	"fmt"

	"github.com/pivotal-cf/brokerapi"
	"github.com/pivotal-cf/cf-redis-broker/brokerconfig"
)

const (
	PlanNameShared    = "shared-vm"
	PlanNameDedicated = "dedicated-vm"
)

type InstanceCredentials struct {
	Host     string
	Port     int
	Password string
}

type InstanceCreator interface {
	Create(instanceID string) error
	Destroy(instanceID string) error
	InstanceExists(instanceID string) (bool, error)
}

type InstanceBinder interface {
	Bind(instanceID string, bindingID string) (InstanceCredentials, error)
	Unbind(instanceID string, bindingID string) error
	InstanceExists(instanceID string) (bool, error)
}

type RedisServiceBroker struct {
	InstanceCreators map[string]InstanceCreator
	InstanceBinders  map[string]InstanceBinder
	Config           brokerconfig.Config
}

func (redisServiceBroker *RedisServiceBroker) Services() []brokerapi.Service {
	planList := []brokerapi.ServicePlan{}
	for _, plan := range redisServiceBroker.plans() {
		planList = append(planList, *plan)
	}

	return []brokerapi.Service{
		brokerapi.Service{
			ID:          redisServiceBroker.Config.RedisConfiguration.ServiceID,
			Name:        redisServiceBroker.Config.RedisConfiguration.ServiceName,
			Description: redisServiceBroker.Config.RedisConfiguration.Description,
			Bindable:    true,
			Plans:       planList,
			Metadata: &brokerapi.ServiceMetadata{
				DisplayName:         redisServiceBroker.Config.RedisConfiguration.DisplayName,
				LongDescription:     redisServiceBroker.Config.RedisConfiguration.LongDescription,
				DocumentationUrl:    redisServiceBroker.Config.RedisConfiguration.DocumentationURL,
				SupportUrl:          redisServiceBroker.Config.RedisConfiguration.SupportURL,
				ImageUrl:            fmt.Sprintf("data:image/png;base64,%s", redisServiceBroker.Config.RedisConfiguration.IconImage),
				ProviderDisplayName: redisServiceBroker.Config.RedisConfiguration.ProviderDisplayName,
			},
			Tags: []string{
				"pivotal",
				"redis",
			},
		},
	}
}

//Provision ...
func (redisServiceBroker *RedisServiceBroker) Provision(instanceID string, serviceDetails brokerapi.ProvisionDetails) error {
	if redisServiceBroker.instanceExists(instanceID) {
		return brokerapi.ErrInstanceAlreadyExists
	}

	if serviceDetails.PlanID == "" {
		return errors.New("plan_id required")
	}

	planIdentifier := ""
	for key, plan := range redisServiceBroker.plans() {
		if plan.ID == serviceDetails.PlanID {
			planIdentifier = key
			break
		}
	}

	if planIdentifier == "" {
		return errors.New("plan_id not recognized")
	}

	instanceCreator, ok := redisServiceBroker.InstanceCreators[planIdentifier]
	if !ok {
		return errors.New("instance creator not found for plan")
	}

	err := instanceCreator.Create(instanceID)
	if err != nil {
		return err
	}

	return nil
}

func (redisServiceBroker *RedisServiceBroker) Deprovision(instanceID string) error {
	for _, instanceCreator := range redisServiceBroker.InstanceCreators {
		instanceExists, _ := instanceCreator.InstanceExists(instanceID)
		if instanceExists {
			return instanceCreator.Destroy(instanceID)
		}
	}
	return brokerapi.ErrInstanceDoesNotExist
}

func (redisServiceBroker *RedisServiceBroker) Bind(instanceID, bindingID string) (interface{}, error) {
	for _, repo := range redisServiceBroker.InstanceBinders {
		instanceExists, _ := repo.InstanceExists(instanceID)
		if instanceExists {
			instanceCredentials, err := repo.Bind(instanceID, bindingID)
			if err != nil {
				return nil, err
			}
			credentialsMap := map[string]interface{}{
				"host":     instanceCredentials.Host,
				"port":     instanceCredentials.Port,
				"password": instanceCredentials.Password,
			}
			return credentialsMap, nil
		}
	}

	return nil, brokerapi.ErrInstanceDoesNotExist
}

func (redisServiceBroker *RedisServiceBroker) Unbind(instanceID, bindingID string) error {
	for _, repo := range redisServiceBroker.InstanceBinders {
		instanceExists, _ := repo.InstanceExists(instanceID)
		if instanceExists {
			err := repo.Unbind(instanceID, bindingID)
			if err != nil {
				return brokerapi.ErrBindingDoesNotExist
			}
			return nil
		}
	}

	return brokerapi.ErrInstanceDoesNotExist
}

func (redisServiceBroker *RedisServiceBroker) plans() map[string]*brokerapi.ServicePlan {
	plans := map[string]*brokerapi.ServicePlan{}

	if redisServiceBroker.Config.SharedEnabled() {
		plans["shared"] = &brokerapi.ServicePlan{
			ID:          redisServiceBroker.Config.RedisConfiguration.SharedVMPlanID,
			Name:        PlanNameShared,
			Description: "This plan provides a single Redis process on a shared VM, which is suitable for development and testing workloads",
			Metadata: &brokerapi.ServicePlanMetadata{
				Bullets: []string{
					"Each instance shares the same VM",
					"Single dedicated Redis process",
					"Suitable for development & testing workloads",
				},
				DisplayName: "Shared-VM",
			},
		}
	}

	if redisServiceBroker.Config.DedicatedEnabled() {
		plans["dedicated"] = &brokerapi.ServicePlan{
			ID:          redisServiceBroker.Config.RedisConfiguration.DedicatedVMPlanID,
			Name:        PlanNameDedicated,
			Description: "This plan provides a single Redis process on a dedicated VM, which is suitable for production workloads",
			Metadata: &brokerapi.ServicePlanMetadata{
				Bullets: []string{
					"Dedicated VM per instance",
					"Single dedicated Redis process",
					"Suitable for production workloads",
				},
				DisplayName: "Dedicated-VM",
			},
		}
	}

	return plans
}

func (redisServiceBroker *RedisServiceBroker) instanceExists(instanceID string) bool {
	for _, instanceCreator := range redisServiceBroker.InstanceCreators {
		instanceExists, _ := instanceCreator.InstanceExists(instanceID)
		if instanceExists {
			return true
		}
	}
	return false
}
