package log

import "code.cloudfoundry.org/lager/v3"

var logger lager.Logger

func Logger() lager.Logger {
	if logger == nil {
		logger = lager.NewLogger("redis-broker")
	}
	return logger
}
