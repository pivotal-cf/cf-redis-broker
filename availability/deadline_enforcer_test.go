package availability_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/cf-redis-broker/availability"
)

var _ = Describe("deadline enforcer", func() {
	Context("when the task does not complete before the deadline", func() {
		It("stops the background task", func() {
			flag := false
			action := func(success chan<- struct{}, terminate <-chan struct{}) {
				select {
				case <-terminate:
					return
				case <-time.After(1 * time.Second):
					flag = true
				}
			}

			enforcer := availability.DeadlineEnforcer{
				Action: action,
			}
			err := enforcer.DoWithin(500 * time.Millisecond)
			Ω(err).Should(HaveOccurred())
			time.Sleep(600 * time.Millisecond)
			Ω(flag).Should(BeFalse())
		})
	})
})
