package broker_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf/brokerapi"
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

	var dedicatedPlanID = "74"

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
					DedicatedVMPlanID:    dedicatedPlanID,
					ServiceInstanceLimit: 3,
					Dedicated: brokerconfig.Dedicated{
						Nodes: []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"},
					},
				},
			},
		}
	})

	Describe(".Provision", func() {
		Context("when the plan is recognized", func() {
			It("creates an instance", func() {
				err := redisBroker.Provision(instanceID, brokerapi.ProvisionDetails{PlanID: sharedPlanID})
				Ω(err).ToNot(HaveOccurred())

				Expect(len(someCreatorAndBinder.createdInstanceIds)).To(Equal(1))
				Expect(someCreatorAndBinder.createdInstanceIds[0]).To(Equal(instanceID))
			})

			Context("when the instance already exists", func() {
				BeforeEach(func() {
					err := redisBroker.Provision(instanceID, brokerapi.ProvisionDetails{PlanID: sharedPlanID})
					Ω(err).ToNot(HaveOccurred())
				})

				It("gives an error when trying to use the same instanceID", func() {
					err := redisBroker.Provision(instanceID, brokerapi.ProvisionDetails{PlanID: sharedPlanID})
					Expect(err).To(Equal(brokerapi.ErrInstanceAlreadyExists))
				})
			})

			Context("when the instance creator returns an error", func() {
				BeforeEach(func() {
					someCreatorAndBinder.createErr = errors.New("something went bad")
				})

				It("returns the same error", func() {
					err := redisBroker.Provision(instanceID, brokerapi.ProvisionDetails{PlanID: sharedPlanID})
					Expect(err).To(MatchError("something went bad"))
				})
			})
		})

		Context("when the plan is not recognized", func() {
			It("returns a suitable error", func() {
				err := redisBroker.Provision(instanceID, brokerapi.ProvisionDetails{PlanID: "not_a_plan_id"})
				Ω(err).To(MatchError("plan_id not recognized"))
			})
		})

		Context("when the plan id is not provided", func() {
			It("returns a suitable error", func() {
				err := redisBroker.Provision(instanceID, brokerapi.ProvisionDetails{})
				Ω(err).To(MatchError("plan_id required"))
			})
		})

		Context("when the plan is recognized, but the broker has not been configured with the appropriate instance creator", func() {
			It("returns a suitable error", func() {
				err := redisBroker.Provision(instanceID, brokerapi.ProvisionDetails{PlanID: dedicatedPlanID})
				Ω(err).To(MatchError("instance creator not found for plan"))
			})
		})
	})

	Describe(".Deprovision", func() {
		BeforeEach(func() {
			err := redisBroker.Provision(instanceID, brokerapi.ProvisionDetails{PlanID: sharedPlanID})
			Ω(err).ToNot(HaveOccurred())
		})

		It("destroys the instance", func() {
			err := redisBroker.Deprovision(instanceID)
			Expect(err).ToNot(HaveOccurred())

			Expect(someCreatorAndBinder.destroyedInstanceIds).To(ContainElement(instanceID))
		})

		It("returns error if instance does not exist", func() {
			err := redisBroker.Deprovision("non-existent")
			Expect(err).To(Equal(brokerapi.ErrInstanceDoesNotExist))
		})

		Context("when the instance creator returns an error", func() {
			BeforeEach(func() {
				someCreatorAndBinder.destroyErr = errors.New("something went bad")
			})

			It("returns the same error", func() {
				err := redisBroker.Deprovision(instanceID)
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

				credentials, err := redisBroker.Bind(instanceID, bindingID)
				Ω(err).NotTo(HaveOccurred())
				expectedCredentials := map[string]interface{}{
					"host":     host,
					"port":     port,
					"password": password,
				}
				Ω(credentials).To(Equal(expectedCredentials))
			})
		})

		Context("when the instance does not exist", func() {
			It("returns brokerapi.InstanceDoesNotExist", func() {
				bindingID := "bindingID"

				_, err := redisBroker.Bind(instanceID, bindingID)
				Ω(err).To(Equal(brokerapi.ErrInstanceDoesNotExist))
			})
		})
	})

	Describe(".Unbind", func() {
		BeforeEach(func() {
			someCreatorAndBinder.Create(instanceID)
			_, err := redisBroker.Bind(instanceID, "EXISTANT-BINDING")
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("returns successfully if binding existed", func() {
			someCreatorAndBinder.bindingExists = true
			err := redisBroker.Unbind(instanceID, "EXISTANT-BINDING")
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("returns brokerapi.ErrBindingDoesNotExist if binding did not exist", func() {
			someCreatorAndBinder.bindingExists = false
			err := redisBroker.Unbind(instanceID, "NON-EXISTANT-BINDING")
			Ω(err).Should(MatchError(brokerapi.ErrBindingDoesNotExist))
		})
	})
})
