package consistency

import (
	"code.cloudfoundry.org/lager"
	"github.com/pivotal-cf/cf-redis-broker/redis"
)

type InconsistencyReporter interface {
	Report(redis.Instance, error)
}

type LogReporter struct {
	logger lager.Logger
}

func NewLogReporter(logger lager.Logger) *LogReporter {
	return &LogReporter{logger}
}

func (l *LogReporter) Report(i redis.Instance, err error) {
	l.logger.Error("consistency_check", err, lager.Data{
		"instance_id":   i.ID,
		"instance_host": i.Host,
		"instance_port": i.Port,
	})
}
