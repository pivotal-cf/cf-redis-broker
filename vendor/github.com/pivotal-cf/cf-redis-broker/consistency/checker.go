package consistency

import (
	"errors"
	"strings"

	"github.com/pivotal-cf/cf-redis-broker/redis"
)

var ErrHasData = errors.New("redis database not empty")

type Checker interface {
	Check() error
}

type Keycounter interface {
	Keycount(host string) (int, error)
}

type InstancesNoDataChecker struct {
	instanceProvider InstancesProvider
	keycounter       Keycounter
	reporter         InconsistencyReporter
}

func NewInstancesNoDataChecker(i InstancesProvider, kc Keycounter, r InconsistencyReporter) *InstancesNoDataChecker {
	return &InstancesNoDataChecker{
		instanceProvider: i,
		keycounter:       kc,
		reporter:         r,
	}
}

func (c *InstancesNoDataChecker) Check() error {
	instances, err := c.instanceProvider.Instances()
	if err != nil {
		return err
	}

	errMessages := []string{}
	for _, i := range instances {
		hasData, err := c.hasData(i)
		if err != nil {
			errMessages = append(errMessages, err.Error())
			continue
		}

		if hasData {
			c.reporter.Report(i, ErrHasData)
		}
	}

	if len(errMessages) > 0 {
		return errors.New(strings.Join(errMessages, "\n"))
	}

	return nil
}

func (c *InstancesNoDataChecker) hasData(i redis.Instance) (bool, error) {
	numKeys, err := c.keycounter.Keycount(i.Host)
	if err != nil {
		return false, err
	}

	return numKeys > 0, nil
}
