package broker

import (
	"errors"

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
			Description: "Redis service to provide a key-value store",
			Bindable:    true,
			Plans:       planList,
			Metadata: brokerapi.ServiceMetadata{
				DisplayName:      "Redis",
				LongDescription:  "",
				DocumentationUrl: "http://docs.pivotal.io/redis/index.html",
				SupportUrl:       "http://support.pivotal.io",
				Listing: brokerapi.ServiceMetadataListing{
					Blurb:    "",
					ImageUrl: "data:image/png;base64,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",
				},
				Provider: brokerapi.ServiceMetadataProvider{
					Name: "Pivotal",
				},
			},
			Tags: []string{
				"pivotal",
				"redis",
			},
		},
	}
}

func (redisServiceBroker *RedisServiceBroker) Provision(instanceID string, serviceDetails brokerapi.ServiceDetails) error {
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
			Metadata: brokerapi.ServicePlanMetadata{
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
			Metadata: brokerapi.ServicePlanMetadata{
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
