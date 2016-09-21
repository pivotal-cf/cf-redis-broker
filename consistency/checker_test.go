package consistency_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf/cf-redis-broker/consistency"
	"github.com/pivotal-cf/cf-redis-broker/consistency/fake"
	"github.com/pivotal-cf/cf-redis-broker/redis"
)

var _ = Describe("InstancesNoDataChecker", func() {
	Describe(".Check", func() {
		var (
			keycounter *fake.Keycounter
			provider   *fake.InstancesProvider
			reporter   *fake.Reporter
			checker    consistency.Checker
		)

		BeforeEach(func() {
			keycounter = new(fake.Keycounter)
			provider = new(fake.InstancesProvider)
			reporter = new(fake.Reporter)

			checker = consistency.NewInstancesNoDataChecker(provider, keycounter, reporter)
		})

		It("does not return an error", func() {
			err := checker.Check()
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when there are multiple instances", func() {
			var instances []redis.Instance

			BeforeEach(func() {
				instances = []redis.Instance{
					redis.Instance{
						Host: "one.example.com",
					},
					redis.Instance{
						Host: "two.example.com",
					},
				}

				provider.InstancesReturns(instances, nil)
			})

			It("does not return an error", func() {
				err := checker.Check()
				Expect(err).NotTo(HaveOccurred())
			})

			Context("when the first redis instance contains keys", func() {
				BeforeEach(func() {
					keycounter.KeycountStub = func(host string) (int, error) {
						if host == instances[0].Host {
							return 7, nil
						}
						return 0, nil
					}
				})

				It("does not return an error", func() {
					err := checker.Check()
					Expect(err).NotTo(HaveOccurred())
				})

				It("reports an inconsistency for the first redis instance", func() {
					checker.Check()

					Expect(reporter.ReportCallCount()).To(Equal(1))

					instance, err := reporter.ReportArgsForCall(0)
					Expect(instance).To(Equal(instances[0]))
					Expect(err).To(Equal(consistency.ErrHasData))
				})

				It("keeps on checking the following instances", func() {
					checker.Check()

					Expect(keycounter.KeycountCallCount()).To(Equal(len(instances)))

					for i, instance := range instances {
						Expect(keycounter.KeycountArgsForCall(i)).To(Equal(instance.Host))
					}
				})
			})

			Context("when getting the keycount for the first redis instance fails", func() {
				var expectedErr = errors.New("some error")

				BeforeEach(func() {
					keycounter.KeycountStub = func(host string) (int, error) {
						if host == instances[0].Host {
							return 0, expectedErr
						}
						return 0, nil
					}
				})

				It("returns the error", func() {
					err := checker.Check()
					Expect(err).To(Equal(expectedErr))
				})

				It("does not report any inconsistency", func() {
					checker.Check()
					Expect(reporter.ReportCallCount()).To(Equal(0))
				})

				It("keeps on checking the following instances", func() {
					checker.Check()

					Expect(keycounter.KeycountCallCount()).To(Equal(len(instances)))

					for i, instance := range instances {
						Expect(keycounter.KeycountArgsForCall(i)).To(Equal(instance.Host))
					}
				})
			})

			Context("when getting the keycount fails for multiple instances", func() {
				var expectedErrs = []error{
					errors.New("error one"),
					errors.New("error two"),
				}

				BeforeEach(func() {
					keycounter.KeycountStub = func(host string) (int, error) {
						return 0, expectedErrs[keycounter.KeycountCallCount()-1]
					}
				})

				It("returns a combined error message", func() {
					err := checker.Check()
					Expect(err.Error()).To(Equal("error one\nerror two"))
				})

				It("does not report any inconsistency", func() {
					checker.Check()
					Expect(reporter.ReportCallCount()).To(Equal(0))
				})
			})

			Context("when getting the instances fails", func() {
				var expectedErr = errors.New("some error")

				BeforeEach(func() {
					provider.InstancesReturns(nil, expectedErr)
				})

				It("returns the error", func() {
					err := checker.Check()
					Expect(err).To(Equal(expectedErr))
				})

				It("does not report any inconsistency", func() {
					checker.Check()
					Expect(reporter.ReportCallCount()).To(Equal(0))
				})
			})
		})
	})
})
