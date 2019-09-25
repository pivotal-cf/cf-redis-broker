package consistency

import (
	"time"

	"code.cloudfoundry.org/lager"
)

type CheckSchedule struct {
	interval time.Duration
	checker  Checker
	logger   lager.Logger
	done     chan struct{}
}

func NewCheckSchedule(checker Checker, interval time.Duration, logger lager.Logger) *CheckSchedule {
	return &CheckSchedule{
		checker:  checker,
		interval: interval,
		logger:   logger,
	}
}

func (cs *CheckSchedule) Start() {
	if cs.done != nil {
		return
	}

	cs.done = make(chan struct{})
	go check(cs.checker, cs.interval, cs.done, cs.logger)
}

func (cs *CheckSchedule) Stop() {
	if cs.done == nil {
		return
	}

	close(cs.done)
	cs.done = nil
}

func check(checker Checker, interval time.Duration, done <-chan struct{}, logger lager.Logger) {
	ticker := time.Tick(interval)

	for {
		select {
		case <-ticker:
			if err := checker.Check(); err != nil {
				logger.Error("check-schedule", err)
			}
		case <-done:
			return
		}
	}
}
