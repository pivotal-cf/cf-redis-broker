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
	go cs.check()
}

func (cs *CheckSchedule) Stop() {
	if cs.done == nil {
		return
	}

	close(cs.done)
	cs.done = nil
}

func (cs *CheckSchedule) check() {
	ticker := time.Tick(cs.interval)

	for {
		select {
		case <-ticker:
			if err := cs.checker.Check(); err != nil {
				cs.logger.Error("check-schedule", err)
			}
		case <-cs.done:
			return
		}
	}
}
