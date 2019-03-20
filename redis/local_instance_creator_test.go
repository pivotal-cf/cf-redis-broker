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

			It("starts a redis instance", func() {
				err := localInstanceCreator.Create(instanceID)
				Expect(err).NotTo(HaveOccurred())

				By("finding a free port", func() {
					Expect(freePortsFound).To(Equal(1))
				})

				By("starting a new redis instance with the correct ID", func() {
					Expect(fakeProcessController.StartAndWaitUntilReadyCallCount()).To(Equal(1))
					instance, _, _, _, _ := fakeProcessController.StartAndWaitUntilReadyArgsForCall(0)
					Expect(instance.ID).To(Equal(instanceID))
				})

				By("calling Unlock on a local repository with the correct instance ID", func() {
					Expect(fakeLocalRepository.UnlockCallCount()).To(Equal(1))
					Expect(fakeLocalRepository.UnlockArgsForCall(0).ID).To(Equal(instanceID))
				})
			})

			Context("when there is not a free port available", func() {
				BeforeEach(func() {
					localInstanceCreator.FindFreePort = func() (int, error) { return 0, errors.New("port not found") }
				})

				It("returns an error", func() {
					err := localInstanceCreator.Create(instanceID)
					Expect(err).To(MatchError("port not found"))
				})
			})

		})

		Context("when the service instance limit has been met", func() {
			BeforeEach(func() {
				fakeLocalRepository.InstanceCountReturns(1, []error{})
			})

			It("does not start a new Redis instance", func() {
				err := localInstanceCreator.Create(instanceID)
				Expect(err).To(MatchError(brokerapi.ErrInstanceLimitMet))

				Expect(fakeProcessController.StartAndWaitUntilReadyCallCount()).To(Equal(0))
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

			It("kills the redis instance", func() {
				err := localInstanceCreator.Destroy(instanceID)
				Expect(err).NotTo(HaveOccurred())

				By("calling lock before stopping redis", func() {
					Expect(fakeLocalRepository.LockCallCount()).To(Equal(1))
					Expect(fakeLocalRepository.LockArgsForCall(0).ID).To(Equal(instanceID))
				})

				By("killing the instance", func() {
					Expect(fakeProcessController.KillCallCount()).To(Equal(1))
					Expect(fakeProcessController.KillArgsForCall(0).ID).To(Equal(instanceID))
				})

				By("deleting the instance data directory", func() {
					Expect(fakeLocalRepository.DeleteCallCount()).To(Equal(1))
					Expect(fakeLocalRepository.DeleteArgsForCall(0)).To(Equal(instanceID))
				})
			})
		})

		Context("when the instance does not exist", func() {
			BeforeEach(func() {
				fakeLocalRepository.FindByIDReturns(nil, errors.New("instance not found"))
			})

			It("returns an appropriate error and does not delete or kill anything", func() {
				err := localInstanceCreator.Destroy("missingInstanceID")
				Expect(err).To(HaveOccurred())

				Expect(fakeProcessController.KillCallCount()).To(Equal(0))
				Expect(fakeLocalRepository.DeleteCallCount()).To(Equal(0))
			})
		})
	})
})
