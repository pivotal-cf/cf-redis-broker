package backup

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("sharedPlan", func() {
	var sharedPlan = &sharedPlan{}
	Describe(".InstanceId", func() {
		var (
			expectedInstanceID = "some-instance-id"
			actualInstanceID   string
			instanceIDErr      error
			redisConfigPath    string
		)

		BeforeEach(func() {
			redisConfigPath = fmt.Sprintf("/var/vcap/store/redis/%s/redis.conf", expectedInstanceID)
		})

		JustBeforeEach(func() {
			actualInstanceID, instanceIDErr = sharedPlan.InstanceID(redisConfigPath, "")
		})

		It("does not return an error", func() {
			Expect(instanceIDErr).ToNot(HaveOccurred())
		})

		It("returns the instance ID based on the redis config path", func() {
			Expect(actualInstanceID).To(Equal(expectedInstanceID))
		})

		Context("when redis config path is a relative path", func() {
			BeforeEach(func() {
				redisConfigPath = fmt.Sprintf("/var/vcap/store/redis/%s/foo/../redis.conf", expectedInstanceID)
			})

			It("does not return an error", func() {
				Expect(instanceIDErr).ToNot(HaveOccurred())
			})

			It("returns the instance ID based on the redis config path", func() {
				Expect(actualInstanceID).To(Equal(expectedInstanceID))
			})
		})

		Context("when redis config path is empty", func() {
			BeforeEach(func() {
				redisConfigPath = ""
			})

			It("returns an error", func() {
				Expect(instanceIDErr).To(HaveOccurred())
			})
		})

		Context("when redis config path is a relative path to a file", func() {
			BeforeEach(func() {
				redisConfigPath = "redis.conf"
			})

			It("returns an error", func() {
				Expect(instanceIDErr).To(HaveOccurred())
			})
		})
	})
})
