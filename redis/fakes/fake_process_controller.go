package fakes

import (
	"time"

	"github.com/pivotal-cf/cf-redis-broker/redis"
)

type FakeProcessController struct {
	StartedInstances  []redis.Instance
	DoOnInstanceStart func()
	KilledInstances   []redis.Instance
	DoOnInstanceStop  func()
}

func (fakeProcessController *FakeProcessController) StartAndWaitUntilReady(instance *redis.Instance, configPath, instanceDataDir, logfilePath string, timeout time.Duration) error {
	fakeProcessController.StartedInstances = append(fakeProcessController.StartedInstances, *instance)
	if fakeProcessController.DoOnInstanceStart != nil {
		fakeProcessController.DoOnInstanceStart()
	}
	return nil
}

func (fakeProcessController *FakeProcessController) Kill(instance *redis.Instance) error {
	fakeProcessController.KilledInstances = append(fakeProcessController.KilledInstances, *instance)
	if fakeProcessController.DoOnInstanceStop != nil {
		fakeProcessController.DoOnInstanceStop()
	}
	return nil
}
