package redis_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pborman/uuid"
	"github.com/pivotal-cf/brokerapi"

	"github.com/pivotal-cf/cf-redis-broker/brokerconfig"
	"github.com/pivotal-cf/cf-redis-broker/redis"
	"github.com/pivotal-cf/cf-redis-broker/redis/fakes"
)

var freePortsFound int

func fakeFreePortFinder() (int, error) {
	freePortsFound++
	return 8080, nil
}

var _ = Describe("Local Redis Creator", func() {
	var (
		instanceID            string
		fakeProcessController *fakes.FakeProcessController
		fakeLocalRepository   *fakes.FakeLocalRepository
		localInstanceCreator  *redis.LocalInstanceCreator
	)

	BeforeEach(func() {
		instanceID = uuid.NewRandom().String()
		fakeProcessController = new(fakes.FakeProcessController)
		fakeLocalRepository = new(fakes.FakeLocalRepository)

		localInstanceCreator = &redis.LocalInstanceCreator{
			FindFreePort:            fakeFreePortFinder,
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
				fakeLocalRepository.InstanceCountReturns(0, []error{errors.New("foo")})
			})

			It("should return an error if unable to retrieve instance count", func() {
				err := localInstanceCreator.Create(instanceID)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when the service instance limit has not been met", func() {
			BeforeEach(func() {
				freePortsFound = 0
			})

			It("finds a free port", func() {
				err := localInstanceCreator.Create(instanceID)
				Expect(err).NotTo(HaveOccurred())
				Expect(freePortsFound).To(Equal(1))
			})

			It("starts a new Redis instance", func() {
				err := localInstanceCreator.Create(instanceID)
				Expect(err).NotTo(HaveOccurred())

				Expect(len(fakeProcessController.StartedInstances)).To(Equal(1))
				Expect(fakeProcessController.StartedInstances[0].ID).To(Equal(instanceID))
			})

			It("calls Unlock on local repository with correct instance ID", func() {
				err := localInstanceCreator.Create(instanceID)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(fakeLocalRepository.UnlockCallCount()).To(Equal(1))
				Expect(fakeLocalRepository.UnlockArgsForCall(0).ID).To(Equal(instanceID))
			})
		})

		Context("when the service instance limit has been met", func() {
			BeforeEach(func() {
				fakeLocalRepository.InstanceCountReturns(1, []error{})
			})

			It("does not start a new Redis instance", func() {
				localInstanceCreator.Create(instanceID)

				Expect(len(fakeProcessController.StartedInstances)).To(Equal(0))
			})

			It("returns an InstanceLimitMet error", func() {
				err := localInstanceCreator.Create(instanceID)
				Expect(err).To(Equal(brokerapi.ErrInstanceLimitMet))
			})
		})
	})

	Describe("destroying a redis instance", func() {
		Context("when the instance exists", func() {
			BeforeEach(func() {
				fakeLocalRepository.FindByIDReturns(
					&redis.Instance{
						ID: instanceID,
					},
					nil,
				)
			})

			JustBeforeEach(func() {
				err := localInstanceCreator.Destroy(instanceID)
				Expect(err).NotTo(HaveOccurred())
			})

			It("calls lock before stopping redis", func() {
				Expect(fakeLocalRepository.LockCallCount()).To(Equal(1))
				Expect(fakeLocalRepository.LockArgsForCall(0).ID).To(Equal(instanceID))
			})

			It("kills the instance", func() {
				Expect(len(fakeProcessController.KilledInstances)).To(Equal(1))
				Expect(fakeProcessController.KilledInstances[0].ID).To(Equal(instanceID))
			})

			It("deletes the instance data directory", func() {
				Expect(fakeLocalRepository.DeleteCallCount()).To(Equal(1))
				Expect(fakeLocalRepository.DeleteArgsForCall(0)).To(Equal(instanceID))
			})
		})

		Context("When the instance does not exist", func() {
			var destroyErr error

			BeforeEach(func() {
				fakeLocalRepository.FindByIDReturns(nil, errors.New("instance not found"))
			})

			JustBeforeEach(func() {
				destroyErr = localInstanceCreator.Destroy("missingInstanceID")
			})

			It("returns an error", func() {
				Expect(destroyErr).To(HaveOccurred())
			})

			It("does not try to kill instance processes", func() {
				Expect(fakeProcessController.KilledInstances).To(BeEmpty())
			})

			It("does not try to delete instances from the instance repository", func() {
				Expect(fakeLocalRepository.DeleteCallCount()).To(Equal(0))
			})
		})
	})
})
