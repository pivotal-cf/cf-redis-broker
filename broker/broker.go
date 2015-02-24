package broker

import (
	"errors"

	"github.com/pivotal-cf/brokerapi"
	"github.com/pivotal-cf/cf-redis-broker/brokerconfig"
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
				DocumentationUrl: "http://docs.pivotal.io/p1-services/Redis.html",
				SupportUrl:       "http://support.pivotal.io",
				Listing: brokerapi.ServiceMetadataListing{
					Blurb:    "",
					ImageUrl: "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAEAAAABACAMAAACdt4HsAAAACXBIWXMAAAsTAAALEwEAmpwYAAAAGXRFWHRTb2Z0d2FyZQBBZG9iZSBJbWFnZVJlYWR5ccllPAAAAJBQTFRF////A3huA3huA3huA3huA3huA3huA3huA3huA3huA3huA3huA3huA3huA3huA3huA3huE4B3IoiAI4mAMpGJQpqSTqCZUqKbYqulcbOtcrOugby3hL24kcTAoc3Jp9DMsdXSwd7byeLgz+Xk0ebk1Ojm2Orp3+7t4O/t6vTz8Pf28vj38vj4+fz8/v/+////J60LdwAAABB0Uk5TABAgMEBQYHCAkKCwwNDg8FTgqMgAAAMZSURBVFjDtVfrmpsgEN0Yk6hrIlSpViorrd1SeuH9365AvQCi0eTr/ApJ5nbmzDC8vPwvOZyi1zQDUrI0jU7BPu1jfAWO3OLjZvXzTLu3cTlsUr+BRcmiuybCFXVt4rSOXAzuyutKEMcr2CBZuKR/ysA2OS+gBzZL9KQ+ALEnfrBLZlkcs30GgFPOw22nPsjs7kjAbkkt/oEH5GIYWEogr1valHAhiYmSkf8fVSsEp0IIgry/JyOCvgoUDRecFPJTSYRgOPf8J1ikEKw6IWgF8lLlAWAtj225SCcXgYLwwSVjLW+c72YoHG3nNRu9IcIxavvDENWcj9YMaGS+NRwSwbhrGcXSQvEmvecSl86syaubAew04lBWj9JGYQ95DjQG74KqswzD9Kf0A+OMhA637SqEUCNUDZD6llf4a4M5kwlgy0Do1AAJXXEsWowbxgbMEKPdN9LxpnANRA4EvQGQy+zxwJ4CAtiMwNoGVEOkloHa/BnVmHZMw1b8ww5+5sS6KqQBa5AS8V5NMCMiatgVU2s0vwQvgIOiRYLu5w/Bm4kvEDNsmBP0k10F14CqQinbh85pq2hEchcDVQYwA1HxZWTTEIgkMpyD6DegOMj6VtSpy9i7gcPrBgoxZdz2zEfUGAgfiCBrBqT932SIPcdyItDOALUg4o9daDUSzGbO+Zfv0t9YqZJQOoKh0CUf21kVUgcDZGZsdbkOxcYgcyZ6XwWJObeGh65LBf1UvniqACoZbotMBpUjV92hdrQwaHNzrimnOvWxL6moZiPJnMkFnxzr4UZVNnBiI7fROcyutZyM2SqHDaXDAWI+sHGUq+9i13iTfDapfbXpb7fZvVK2zvwtqf966m8Wz2qm6jZWsmJWi7tD2R6rRv8qryVUhGaV/34N72wHKm8prLq7IQSL+w3CuFhcEIx9MXpkwUjMFeW6Xz87PLWkWQnsXjO9y2r8BAAPbHpX386fPKm/w0Ky9OY4P4ifUc0NK/P6q+n+oym5924L0tUNO9zwcAyTp9R1FBdPb9ziXe/n4BwbuVyTc/DQEz7UsgrbX7DH4eYGB7x1AAAAAElFTkSuQmCC",
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

	return instanceCreator.Create(instanceID)
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
			ID:          "C210CA06-E7E5-4F5D-A5AA-7A2C51CC290E",
			Name:        "shared-vm",
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
			ID:          "74E8984C-5F8C-11E4-86BE-07807B3B2589",
			Name:        "dedicated-vm",
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
