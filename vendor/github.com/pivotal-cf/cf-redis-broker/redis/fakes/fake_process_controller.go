// Code generated by counterfeiter. DO NOT EDIT.
package fakes

import (
	"sync"
	"time"

	"github.com/pivotal-cf/cf-redis-broker/redis"
)

type FakeProcessController struct {
	KillStub        func(*redis.Instance) error
	killMutex       sync.RWMutex
	killArgsForCall []struct {
		arg1 *redis.Instance
	}
	killReturns struct {
		result1 error
	}
	killReturnsOnCall map[int]struct {
		result1 error
	}
	StartAndWaitUntilReadyStub        func(*redis.Instance, string, string, string, time.Duration) error
	startAndWaitUntilReadyMutex       sync.RWMutex
	startAndWaitUntilReadyArgsForCall []struct {
		arg1 *redis.Instance
		arg2 string
		arg3 string
		arg4 string
		arg5 time.Duration
	}
	startAndWaitUntilReadyReturns struct {
		result1 error
	}
	startAndWaitUntilReadyReturnsOnCall map[int]struct {
		result1 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeProcessController) Kill(arg1 *redis.Instance) error {
	fake.killMutex.Lock()
	ret, specificReturn := fake.killReturnsOnCall[len(fake.killArgsForCall)]
	fake.killArgsForCall = append(fake.killArgsForCall, struct {
		arg1 *redis.Instance
	}{arg1})
	fake.recordInvocation("Kill", []interface{}{arg1})
	fake.killMutex.Unlock()
	if fake.KillStub != nil {
		return fake.KillStub(arg1)
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.killReturns
	return fakeReturns.result1
}

func (fake *FakeProcessController) KillCallCount() int {
	fake.killMutex.RLock()
	defer fake.killMutex.RUnlock()
	return len(fake.killArgsForCall)
}

func (fake *FakeProcessController) KillCalls(stub func(*redis.Instance) error) {
	fake.killMutex.Lock()
	defer fake.killMutex.Unlock()
	fake.KillStub = stub
}

func (fake *FakeProcessController) KillArgsForCall(i int) *redis.Instance {
	fake.killMutex.RLock()
	defer fake.killMutex.RUnlock()
	argsForCall := fake.killArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeProcessController) KillReturns(result1 error) {
	fake.killMutex.Lock()
	defer fake.killMutex.Unlock()
	fake.KillStub = nil
	fake.killReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeProcessController) KillReturnsOnCall(i int, result1 error) {
	fake.killMutex.Lock()
	defer fake.killMutex.Unlock()
	fake.KillStub = nil
	if fake.killReturnsOnCall == nil {
		fake.killReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.killReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeProcessController) StartAndWaitUntilReady(arg1 *redis.Instance, arg2 string, arg3 string, arg4 string, arg5 time.Duration) error {
	fake.startAndWaitUntilReadyMutex.Lock()
	ret, specificReturn := fake.startAndWaitUntilReadyReturnsOnCall[len(fake.startAndWaitUntilReadyArgsForCall)]
	fake.startAndWaitUntilReadyArgsForCall = append(fake.startAndWaitUntilReadyArgsForCall, struct {
		arg1 *redis.Instance
		arg2 string
		arg3 string
		arg4 string
		arg5 time.Duration
	}{arg1, arg2, arg3, arg4, arg5})
	fake.recordInvocation("StartAndWaitUntilReady", []interface{}{arg1, arg2, arg3, arg4, arg5})
	fake.startAndWaitUntilReadyMutex.Unlock()
	if fake.StartAndWaitUntilReadyStub != nil {
		return fake.StartAndWaitUntilReadyStub(arg1, arg2, arg3, arg4, arg5)
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.startAndWaitUntilReadyReturns
	return fakeReturns.result1
}

func (fake *FakeProcessController) StartAndWaitUntilReadyCallCount() int {
	fake.startAndWaitUntilReadyMutex.RLock()
	defer fake.startAndWaitUntilReadyMutex.RUnlock()
	return len(fake.startAndWaitUntilReadyArgsForCall)
}

func (fake *FakeProcessController) StartAndWaitUntilReadyCalls(stub func(*redis.Instance, string, string, string, time.Duration) error) {
	fake.startAndWaitUntilReadyMutex.Lock()
	defer fake.startAndWaitUntilReadyMutex.Unlock()
	fake.StartAndWaitUntilReadyStub = stub
}

func (fake *FakeProcessController) StartAndWaitUntilReadyArgsForCall(i int) (*redis.Instance, string, string, string, time.Duration) {
	fake.startAndWaitUntilReadyMutex.RLock()
	defer fake.startAndWaitUntilReadyMutex.RUnlock()
	argsForCall := fake.startAndWaitUntilReadyArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3, argsForCall.arg4, argsForCall.arg5
}

func (fake *FakeProcessController) StartAndWaitUntilReadyReturns(result1 error) {
	fake.startAndWaitUntilReadyMutex.Lock()
	defer fake.startAndWaitUntilReadyMutex.Unlock()
	fake.StartAndWaitUntilReadyStub = nil
	fake.startAndWaitUntilReadyReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeProcessController) StartAndWaitUntilReadyReturnsOnCall(i int, result1 error) {
	fake.startAndWaitUntilReadyMutex.Lock()
	defer fake.startAndWaitUntilReadyMutex.Unlock()
	fake.StartAndWaitUntilReadyStub = nil
	if fake.startAndWaitUntilReadyReturnsOnCall == nil {
		fake.startAndWaitUntilReadyReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.startAndWaitUntilReadyReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeProcessController) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.killMutex.RLock()
	defer fake.killMutex.RUnlock()
	fake.startAndWaitUntilReadyMutex.RLock()
	defer fake.startAndWaitUntilReadyMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeProcessController) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ redis.ProcessController = new(FakeProcessController)