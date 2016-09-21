package consistency_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/st3v/glager"

	"github.com/pivotal-cf/cf-redis-broker/consistency"
	"github.com/pivotal-cf/cf-redis-broker/redis"
)

var _ = Describe("LogReporter", func() {
	Describe(".Report", func() {
		var (
			logger      = NewLogger("log-reporter-test")
			reporter    = consistency.NewLogReporter(logger)
			expectedErr = errors.New("some error")
			instance    = redis.Instance{
				ID:   "my-instance",
				Host: "my-host",
				Port: 1234,
			}
		)

		It("logs the inconsistency error and instance details", func() {
			reporter.Report(instance, expectedErr)
			Expect(logger).To(HaveLogged(
				Error(expectedErr, Data(
					"instance_id", "my-instance",
					"instance_host", "my-host",
					"instance_port", 1234,
				)),
			))
		})
	})

})
