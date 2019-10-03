package broker_test

import (
	"errors"
	brokerapi "github.com/pivotal-cf/brokerapi/domain"
	brokerapiresponses "github.com/pivotal-cf/brokerapi/domain/apiresponses"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf/cf-redis-broker/broker"
	"github.com/pivotal-cf/cf-redis-broker/brokerconfig"
)

type fakeInstanceCreatorAndBinder struct {
	createErr            error
	createdInstanceIds   []string
	destroyErr           error
	destroyedInstanceIds []string
	instanceCredentials  broker.InstanceCredentials
	bindingExists        bool
}

func (fakeInstanceCreatorAndBinder *fakeInstanceCreatorAndBinder) Create(instanceID string) error {
	if fakeInstanceCreatorAndBinder.createErr != nil {
		return fakeInstanceCreatorAndBinder.createErr
	}
	fakeInstanceCreatorAndBinder.createdInstanceIds = append(fakeInstanceCreatorAndBinder.createdInstanceIds, instanceID)
	return nil
}

func (fakeInstanceCreatorAndBinder *fakeInstanceCreatorAndBinder) Destroy(instanceID string) error {
	if fakeInstanceCreatorAndBinder.destroyErr != nil {
		return fakeInstanceCreatorAndBinder.destroyErr
	}
	fakeInstanceCreatorAndBinder.destroyedInstanceIds = append(fakeInstanceCreatorAndBinder.destroyedInstanceIds, instanceID)
	return nil
}

func (fakeInstanceCreatorAndBinder *fakeInstanceCreatorAndBinder) Bind(instanceID string, bindingID string) (broker.InstanceCredentials, error) {
	return fakeInstanceCreatorAndBinder.instanceCredentials, nil
}

func (fakeInstanceCreatorAndBinder *fakeInstanceCreatorAndBinder) Unbind(instanceID string, bindingID string) error {
	if !fakeInstanceCreatorAndBinder.bindingExists {
		return errors.New("unbind error")
	}
	return nil
}

func (fakeInstanceCreatorAndBinder *fakeInstanceCreatorAndBinder) InstanceExists(instanceID string) (bool, error) {
	for _, existingInstanceID := range fakeInstanceCreatorAndBinder.createdInstanceIds {
		if instanceID == existingInstanceID {
			return true, nil
		}
	}
	return false, nil
}

var _ = Describe("Redis service broker", func() {

	const instanceID = "instanceID"

	var redisBroker *broker.RedisServiceBroker

	var someCreatorAndBinder *fakeInstanceCreatorAndBinder

	var sharedPlanID = "C210CA06-E7E5-4F5D-A5AA-7A2C51CC290E"
	var planName = "shared"

	var host = "an_host"
	var port = 1234
	var password = "big_secret"

	BeforeEach(func() {
		someCreatorAndBinder = &fakeInstanceCreatorAndBinder{
			instanceCredentials: broker.InstanceCredentials{
				Host:     host,
				Port:     port,
				Password: password,
			},
		}

		redisBroker = &broker.RedisServiceBroker{
			InstanceCreators: map[string]broker.InstanceCreator{
				planName: someCreatorAndBinder,
			},
			InstanceBinders: map[string]broker.InstanceBinder{
				planName: someCreatorAndBinder,
			},
			Config: brokerconfig.Config{
				RedisConfiguration: brokerconfig.ServiceConfiguration{
					SharedVMPlanID:       sharedPlanID,
					ServiceInstanceLimit: 3,
				},
			},
		}
	})

	Describe(".Provision", func() {
		Context("when the plan is recognized", func() {
			It("creates an instance", func() {
				_, err := redisBroker.Provision(nil, instanceID, brokerapi.ProvisionDetails{PlanID: sharedPlanID}, false)
				Expect(err).NotTo(HaveOccurred())

				Expect(len(someCreatorAndBinder.createdInstanceIds)).To(Equal(1))
				Expect(someCreatorAndBinder.createdInstanceIds[0]).To(Equal(instanceID))
			})

			Context("when the instance already exists", func() {
				BeforeEach(func() {
					_, err := redisBroker.Provision(nil, instanceID, brokerapi.ProvisionDetails{PlanID: sharedPlanID}, false)
					Expect(err).NotTo(HaveOccurred())
				})

				It("gives an error when trying to use the same instanceID", func() {
					_, err := redisBroker.Provision(nil, instanceID, brokerapi.ProvisionDetails{PlanID: sharedPlanID}, false)
					Expect(err).To(Equal(brokerapiresponses.ErrInstanceAlreadyExists))
				})
			})

			Context("when the instance creator returns an error", func() {
				BeforeEach(func() {
					someCreatorAndBinder.createErr = errors.New("something went bad")
				})

				It("returns the same error", func() {
					_, err := redisBroker.Provision(nil, instanceID, brokerapi.ProvisionDetails{PlanID: sharedPlanID}, false)
					Expect(err).To(MatchError("something went bad"))
				})
			})
		})

		Context("when the plan is not recognized", func() {
			It("returns a suitable error", func() {
				_, err := redisBroker.Provision(nil, instanceID, brokerapi.ProvisionDetails{PlanID: "not_a_plan_id"}, false)
				Expect(err).To(MatchError("plan_id not recognized"))
			})
		})

		Context("when the plan id is not provided", func() {
			It("returns a suitable error", func() {
				_, err := redisBroker.Provision(nil, instanceID, brokerapi.ProvisionDetails{}, false)
				Expect(err).To(MatchError("plan_id required"))
			})
		})
	})

	Describe(".Deprovision", func() {
		BeforeEach(func() {
			_, err := redisBroker.Provision(nil, instanceID, brokerapi.ProvisionDetails{PlanID: sharedPlanID}, false)
			Expect(err).NotTo(HaveOccurred())
		})

		It("destroys the instance", func() {
			_, err := redisBroker.Deprovision(nil, instanceID, brokerapi.DeprovisionDetails{}, false)
			Expect(err).NotTo(HaveOccurred())

			Expect(someCreatorAndBinder.destroyedInstanceIds).To(ContainElement(instanceID))
		})

		It("returns error if instance does not exist", func() {
			_, err := redisBroker.Deprovision(nil, "non-existent", brokerapi.DeprovisionDetails{}, false)
			Expect(err).To(Equal(brokerapiresponses.ErrInstanceDoesNotExist))
		})

		Context("when the instance creator returns an error", func() {
			BeforeEach(func() {
				someCreatorAndBinder.destroyErr = errors.New("something went bad")
			})

			It("returns the same error", func() {
				_, err := redisBroker.Deprovision(nil, instanceID, brokerapi.DeprovisionDetails{}, false)
				Expect(err).To(MatchError("something went bad"))
			})
		})
	})

	Describe(".Bind", func() {
		Context("when the instance exists", func() {
			BeforeEach(func() {
				someCreatorAndBinder.Create(instanceID)
			})

			It("returns credentials", func() {
				bindingID := "bindingID"

				credentials, err := redisBroker.Bind(nil, instanceID, bindingID, brokerapi.BindDetails{}, false)
				Expect(err).NotTo(HaveOccurred())

				expectedCredentials := brokerapi.Binding{
					Credentials: map[string]interface{}{
						"host":     host,
						"port":     port,
						"password": password,
					},
					SyslogDrainURL:  "",
					RouteServiceURL: "",
				}

				Expect(credentials).To(Equal(expectedCredentials))
			})
		})

		Context("when the instance does not exist", func() {
			It("returns brokerapi.InstanceDoesNotExist", func() {
				bindingID := "bindingID"

				_, err := redisBroker.Bind(nil, instanceID, bindingID, brokerapi.BindDetails{}, false)
				Expect(err).To(Equal(brokerapiresponses.ErrInstanceDoesNotExist))
			})
		})
	})

	Describe(".Unbind", func() {
		BeforeEach(func() {
			someCreatorAndBinder.Create(instanceID)
			_, err := redisBroker.Bind(nil, instanceID, "EXISTANT-BINDING", brokerapi.BindDetails{}, false)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns successfully if binding existed", func() {
			someCreatorAndBinder.bindingExists = true
			_, err := redisBroker.Unbind(nil, instanceID, "EXISTANT-BINDING", brokerapi.UnbindDetails{}, false)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns brokerapi.ErrBindingDoesNotExist if binding did not exist", func() {
			someCreatorAndBinder.bindingExists = false
			_, err := redisBroker.Unbind(nil, instanceID, "NON-EXISTANT-BINDING", brokerapi.UnbindDetails{}, false)
			Expect(err).To(MatchError(brokerapiresponses.ErrBindingDoesNotExist))
		})
	})
})
