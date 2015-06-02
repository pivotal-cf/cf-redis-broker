package shared_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-cf/cf-redis-broker/plan"
	"github.com/pivotal-cf/cf-redis-broker/plan/shared"
	"github.com/pivotal-golang/lager"
	. "github.com/st3v/glager"
)

var _ = Describe("shared.InstanceIDProvider", func() {

	Describe(".InstanceId", func() {
		var (
			idProvider         plan.IDProvider
			expectedInstanceID = "some-instance-id"
			actualInstanceID   string
			instanceIDErr      error
			redisConfigPath    string
			log                *gbytes.Buffer
		)

		BeforeEach(func() {
			log = gbytes.NewBuffer()
			logger := lager.NewLogger("provider")
			logger.RegisterSink(lager.NewWriterSink(log, lager.DEBUG))

			redisConfigPath = fmt.Sprintf("/var/vcap/store/redis/%s/redis.conf", expectedInstanceID)

			idProvider = shared.InstanceIDProvider(logger)
		})

		JustBeforeEach(func() {
			actualInstanceID, instanceIDErr = idProvider.InstanceID(redisConfigPath, "")
		})

		It("does not return an error", func() {
			Expect(instanceIDErr).ToNot(HaveOccurred())
		})

		It("returns the instance ID based on the redis config path", func() {
			Expect(actualInstanceID).To(Equal(expectedInstanceID))
		})

		It("provides logging", func() {
			Expect(log).To(ContainSequence(
				Info(
					Action("provider.shared-instance-id"),
					Data("event", "starting", "path", redisConfigPath),
				),
				Info(
					Action("provider.shared-instance-id"),
					Data("event", "done", "path", redisConfigPath, "instance_id", expectedInstanceID),
				),
			))
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
			It("logs the error", func() {
				Expect(log).To(ContainSequence(
					Info(
						Action("provider.shared-instance-id"),
						Data("event", "starting", "path", redisConfigPath),
					),
					Error(
						instanceIDErr,
						Action("provider.shared-instance-id"),
						Data("event", "failed", "path", redisConfigPath),
					),
				))
			})
		})
	})
})
