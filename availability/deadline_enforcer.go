package availability

import (
	"errors"
	"time"
)

type DeadlineEnforcer struct {
	Action func(success chan<- struct{}, terminate <-chan struct{})
}

func (deadlineEnforcer DeadlineEnforcer) DoWithin(duration time.Duration) error {
	success := make(chan struct{})
	terminate := make(chan struct{})

	go deadlineEnforcer.Action(success, terminate)
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-success:
		return nil
	case <-timer.C:
		close(terminate)
		return errors.New("timeout")
	}
}
