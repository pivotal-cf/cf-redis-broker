package consistency

import (
	"time"

	"code.cloudfoundry.org/lager"

	"github.com/pivotal-cf/cf-redis-broker/redis"
)

var schedule *CheckSchedule

func KeepVerifying(agentClient *redis.RemoteAgentClient, statefilePath string, interval time.Duration, logger lager.Logger) {
	if schedule != nil {
		return
	}

	checker := NewInstancesNoDataChecker(
		NewStateFileAvailableInstances(statefilePath),
		agentClient,
		NewLogReporter(logger),
	)

	schedule = NewCheckSchedule(checker, interval, logger)
	schedule.Start()

	logger.Info("consistency.keep-verifying", lager.Data{
		"message":  "started",
		"interval": interval.String(),
	})
}

func StopVerifying() {
	if schedule == nil {
		return
	}

	schedule.Stop()
	schedule = nil
}
