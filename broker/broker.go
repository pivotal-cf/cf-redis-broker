package broker

import (
	"context"
	"errors"
	"fmt"
	brokerapi "github.com/pivotal-cf/brokerapi/domain"
	brokerapiresponses "github.com/pivotal-cf/brokerapi/domain/apiresponses"

	"github.com/pivotal-cf/cf-redis-broker/brokerconfig"
)

const (
	PlanNameShared = "shared-vm"
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

func (redisServiceBroker *RedisServiceBroker) Services(ctx context.Context) ([]brokerapi.Service, error) {
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
	}, nil
}

//Provision ...
func (redisServiceBroker *RedisServiceBroker) Provision(ctx context.Context, instanceID string, serviceDetails brokerapi.ProvisionDetails, asyncAllowed bool) (spec brokerapi.ProvisionedServiceSpec, err error) {
	spec = brokerapi.ProvisionedServiceSpec{}

	if redisServiceBroker.instanceExists(instanceID) {
		return spec, brokerapiresponses.ErrInstanceAlreadyExists
	}

	if serviceDetails.PlanID == "" {
		return spec, errors.New("plan_id required")
	}

	planIdentifier := ""
	for key, plan := range redisServiceBroker.plans() {
		if plan.ID == serviceDetails.PlanID {
			planIdentifier = key
			break
		}
	}

	if planIdentifier == "" {
		return spec, errors.New("plan_id not recognized")
	}

	instanceCreator, ok := redisServiceBroker.InstanceCreators[planIdentifier]
	if !ok {
		return spec, errors.New("instance creator not found for plan")
	}

	err = instanceCreator.Create(instanceID)
	if err != nil {
		return spec, err
	}

	return spec, nil
}

func (redisServiceBroker *RedisServiceBroker) Deprovision(ctx context.Context, instanceID string, details brokerapi.DeprovisionDetails, asyncAllowed bool) (brokerapi.DeprovisionServiceSpec, error) {
	spec := brokerapi.DeprovisionServiceSpec{}

	for _, instanceCreator := range redisServiceBroker.InstanceCreators {
		instanceExists, _ := instanceCreator.InstanceExists(instanceID)
		if instanceExists {
			return spec, instanceCreator.Destroy(instanceID)
		}
	}
	return spec, brokerapiresponses.ErrInstanceDoesNotExist
}

func (redisServiceBroker *RedisServiceBroker) Bind(ctx context.Context, instanceID, bindingID string, details brokerapi.BindDetails, asyncAllowed bool) (brokerapi.Binding, error) {
	binding := brokerapi.Binding{}

	for _, repo := range redisServiceBroker.InstanceBinders {
		instanceExists, _ := repo.InstanceExists(instanceID)
		if instanceExists {
			instanceCredentials, err := repo.Bind(instanceID, bindingID)
			if err != nil {
				return binding, err
			}
			credentialsMap := map[string]interface{}{
				"host":     instanceCredentials.Host,
				"port":     instanceCredentials.Port,
				"password": instanceCredentials.Password,
			}

			binding.Credentials = credentialsMap
			return binding, nil
		}
	}
	return brokerapi.Binding{}, brokerapiresponses.ErrInstanceDoesNotExist
}

func (redisServiceBroker *RedisServiceBroker) Unbind(ctx context.Context, instanceID, bindingID string, details brokerapi.UnbindDetails, asyncAllowed bool) (brokerapi.UnbindSpec, error) {
	for _, repo := range redisServiceBroker.InstanceBinders {
		instanceExists, _ := repo.InstanceExists(instanceID)
		if instanceExists {
			err := repo.Unbind(instanceID, bindingID)
			if err != nil {
				return brokerapi.UnbindSpec{}, brokerapiresponses.ErrBindingDoesNotExist
			}
			return brokerapi.UnbindSpec{}, nil
		}
	}

	return brokerapi.UnbindSpec{}, brokerapiresponses.ErrInstanceDoesNotExist
}

func (redisServiceBroker *RedisServiceBroker) plans() map[string]*brokerapi.ServicePlan {
	plans := map[string]*brokerapi.ServicePlan{}

	if redisServiceBroker.Config.SharedEnabled() {
		plans["shared"] = &brokerapi.ServicePlan{
			ID:          redisServiceBroker.Config.RedisConfiguration.SharedVMPlanID,
			Name:        PlanNameShared,
			Description: "This plan provides a Redis server on a shared VM configured for data persistence.",
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

// LastOperation ...
// If the broker provisions asynchronously, the Cloud Controller will poll this endpoint
// for the status of the provisioning operation.
func (redisServiceBroker *RedisServiceBroker) LastOperation(ctx context.Context, instanceID string, details brokerapi.PollDetails) (brokerapi.LastOperation, error) {
	return brokerapi.LastOperation{}, errors.New("not implemented")
}

func (redisServiceBroker *RedisServiceBroker) Update(cxt context.Context, instanceID string, details brokerapi.UpdateDetails, asyncAllowed bool) (brokerapi.UpdateServiceSpec, error) {
	return brokerapi.UpdateServiceSpec{}, errors.New("not implemented")
}

func (redisServiceBroker *RedisServiceBroker) GetBinding(ctx context.Context, instanceID, bindingID string) (brokerapi.GetBindingSpec, error) {
	return brokerapi.GetBindingSpec{}, errors.New("not implemented")
}

func (redisServiceBroker *RedisServiceBroker) GetInstance(ctx context.Context, instanceID string) (brokerapi.GetInstanceDetailsSpec, error) {
	return brokerapi.GetInstanceDetailsSpec{}, errors.New("not implemented")
}

func (redisServiceBroker *RedisServiceBroker) LastBindingOperation(ctx context.Context, instanceID, bindingID string, details brokerapi.PollDetails) (brokerapi.LastOperation, error) {
	return brokerapi.LastOperation{}, errors.New("not implemented")
}
