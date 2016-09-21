package consistency_test

import (
	"errors"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/st3v/glager"

	"github.com/pivotal-cf/cf-redis-broker/consistency"
	"github.com/pivotal-cf/cf-redis-broker/consistency/fake"
)

var _ = Describe("CheckSchedule", func() {
	var (
		schedule *consistency.CheckSchedule
		checker  *fake.Checker
		logger   *TestLogger

		interval = 10 * time.Millisecond
	)

	BeforeEach(func() {
		checker = new(fake.Checker)
		logger = NewLogger("consistency_test")
		schedule = consistency.NewCheckSchedule(checker, interval, logger)
	})

	AfterEach(func() {
		schedule.Stop()
	})

	Context("when the schedule has not been started", func() {
		It("does not invoke the checker", func() {
			Consistently(checker.CheckCallCount).Should(Equal(0))
		})
	})

	Context("when the schedule has been started", func() {
		BeforeEach(func() {
			schedule.Start()
		})

		It("checks regularly", func() {
			Eventually(checker.CheckCallCount).Should(BeNumerically(">", 10))
		})

		Context("and a check fails", func() {
			var expectedErr = errors.New("some error")

			BeforeEach(func() {
				checker.CheckStub = func() error {
					if checker.CheckCallCount() == 3 {
						return expectedErr
					}
					return nil
				}
			})

			It("logs the corresponding error", func() {
				Eventually(logger).Should(HaveLogged(
					Error(expectedErr),
				))
			})

			It("keeps on checking regularly", func() {
				Eventually(checker.CheckCallCount).Should(BeNumerically(">", 10))
			})
		})
	})
})
