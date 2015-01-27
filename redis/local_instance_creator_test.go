package redis_test

import (
	"errors"

	"code.google.com/p/go-uuid/uuid"

	"github.com/pivotal-cf/brokerapi"
	"github.com/pivotal-cf/cf-redis-broker/brokerconfig"
	"github.com/pivotal-cf/cf-redis-broker/redis"
	"github.com/pivotal-cf/cf-redis-broker/redis/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var freePortsFound int

func fakeFreePortFinder() (int, error) {
	freePortsFound++
	return 8080, nil
}

var _ = Describe("Local Redis Creator", func() {

	var instanceID string
	var fakeProcessController *fakes.FakeProcessController
	var fakeCredentialGenerator *fakes.FakeCredentialGenerator
	var fakeLocalRepository *fakes.FakeLocalRepository
	var localInstanceCreator *redis.LocalInstanceCreator

	BeforeEach(func() {
		instanceID = uuid.NewRandom().String()
		fakeProcessController = &fakes.FakeProcessController{}

		fakeLocalRepository = &fakes.FakeLocalRepository{
			DeletedInstanceIds: []string{},
			CreatedInstances:   []*redis.Instance{},
			Instances:          []*redis.Instance{},
		}

		fakeCredentialGenerator = &fakes.FakeCredentialGenerator{}

		localInstanceCreator = &redis.LocalInstanceCreator{
			FindFreePort:            fakeFreePortFinder,
			CredentialsGenerator:    fakeCredentialGenerator,
			ProcessController:       fakeProcessController,
			LocalInstanceRepository: fakeLocalRepository,
			RedisConfiguration: brokerconfig.ServiceConfiguration{
				ServiceInstanceLimit: 1,
			},
		}
	})

	Describe("Create", func() {
		Context("when retrieving the number of instances fails", func() {
			BeforeEach(func() {
				localInstanceCreator.LocalInstanceRepository = &fakes.FakeLocalRepository{
					DeletedInstanceIds: []string{},
					CreatedInstances:   []*redis.Instance{},
					Instances:          []*redis.Instance{},
					InstanceCountErr:   errors.New("could not retrieve instance count"),
				}
			})

			It("should return an error if unable to retrieve instance count", func() {
				err := localInstanceCreator.Create(instanceID)
				Ω(err).To(HaveOccurred())
			})
		})

		Context("when the service instance limit has not been met", func() {
			BeforeEach(func() {
				freePortsFound = 0
			})

			It("finds a free port", func() {
				err := localInstanceCreator.Create(instanceID)
				Ω(err).ToNot(HaveOccurred())

				Ω(freePortsFound).To(Equal(1))
			})

			It("starts a new Redis instance", func() {
				err := localInstanceCreator.Create(instanceID)
				Ω(err).ToNot(HaveOccurred())

				Ω(len(fakeProcessController.StartedInstances)).To(Equal(1))
				Ω(fakeProcessController.StartedInstances[0].ID).To(Equal(instanceID))
			})

			It("deletes the lock file in the instance data directory after redis has been started", func() {
				fakeProcessController.DoOnInstanceStart = func() {
					Ω(fakeLocalRepository.UnlockedInstances).Should(BeEmpty())
				}
				err := localInstanceCreator.Create(instanceID)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(fakeLocalRepository.UnlockedInstances).Should(HaveLen(1))
				Ω(fakeLocalRepository.UnlockedInstances[0].ID).Should(Equal(instanceID))
			})

		})

		Context("when the service instance limit has been met", func() {
			BeforeEach(func() {
				fakeLocalRepository.Instances = []*redis.Instance{
					&redis.Instance{
						ID:       "1",
						Port:     1234,
						Host:     "whatever",
						Password: "whatever",
					},
				}
			})

			It("does not start a new Redis instance", func() {
				localInstanceCreator.Create(instanceID)

				Ω(len(fakeProcessController.StartedInstances)).To(Equal(0))
			})

			It("returns an InstanceLimitMet error", func() {
				err := localInstanceCreator.Create(instanceID)
				Ω(err).To(Equal(brokerapi.ErrInstanceLimitMet))
			})
		})
	})

	Describe("destroying a redis instance", func() {
		Context("when the instance exists", func() {
			BeforeEach(func() {
				localInstanceCreator.Create(instanceID)
			})

			It("calls lock before stopping redis", func() {
				Ω(fakeLocalRepository.LockedInstances).To(BeEmpty())
				fakeProcessController.DoOnInstanceStop = func() {
					Ω(fakeLocalRepository.LockedInstances).To(HaveLen(1))
					Ω(fakeLocalRepository.LockedInstances[0].ID).To(Equal(instanceID))
				}
				err := localInstanceCreator.Destroy(instanceID)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("kills the instance", func() {
				err := localInstanceCreator.Destroy(instanceID)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(len(fakeProcessController.KilledInstances)).To(Equal(1))
				Ω(fakeProcessController.KilledInstances[0].ID).To(Equal(instanceID))
			})

			It("deletes the instance data directory", func() {
				localInstanceCreator.Destroy(instanceID)
				Ω(fakeLocalRepository.DeletedInstanceIds).To(Equal([]string{
					instanceID,
				}))
			})
		})

		Context("When the instance does not exist", func() {
			var destroyErr error
			BeforeEach(func() {
				missingInstanceID := "missingInstanceID"
				destroyErr = localInstanceCreator.Destroy(missingInstanceID)
			})

			It("returns an error", func() {
				Ω(destroyErr).To(HaveOccurred())
			})

			It("does not try to kill instance processes", func() {
				Ω(fakeProcessController.KilledInstances).To(BeEmpty())
			})

			It("does not try to delete instances from the instance repository", func() {
				Ω(fakeLocalRepository.DeletedInstanceIds).To(BeEmpty())
			})
		})
	})
})
