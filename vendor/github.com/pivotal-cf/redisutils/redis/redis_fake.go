// This file was generated by counterfeiter
// counterfeiter -o redis/redis_fake.go --fake-name Fake redis/redis.go Redis

package redis

import (
	"sync"

	redigo "github.com/gomodule/redigo/redis"
)

type Fake struct {
	DialStub        func(string, string, ...redigo.DialOption) (redigo.Conn, error)
	dialMutex       sync.RWMutex
	dialArgsForCall []struct {
		arg1 string
		arg2 string
		arg3 []redigo.DialOption
	}
	dialReturns struct {
		result1 redigo.Conn
		result2 error
	}
	dialReturnsOnCall map[int]struct {
		result1 redigo.Conn
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *Fake) Dial(arg1 string, arg2 string, arg3 ...redigo.DialOption) (redigo.Conn, error) {
	fake.dialMutex.Lock()
	ret, specificReturn := fake.dialReturnsOnCall[len(fake.dialArgsForCall)]
	fake.dialArgsForCall = append(fake.dialArgsForCall, struct {
		arg1 string
		arg2 string
		arg3 []redigo.DialOption
	}{arg1, arg2, arg3})
	fake.recordInvocation("Dial", []interface{}{arg1, arg2, arg3})
	fake.dialMutex.Unlock()
	if fake.DialStub != nil {
		return fake.DialStub(arg1, arg2, arg3...)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fake.dialReturns.result1, fake.dialReturns.result2
}

func (fake *Fake) DialCallCount() int {
	fake.dialMutex.RLock()
	defer fake.dialMutex.RUnlock()
	return len(fake.dialArgsForCall)
}

func (fake *Fake) DialArgsForCall(i int) (string, string, []redigo.DialOption) {
	fake.dialMutex.RLock()
	defer fake.dialMutex.RUnlock()
	return fake.dialArgsForCall[i].arg1, fake.dialArgsForCall[i].arg2, fake.dialArgsForCall[i].arg3
}

func (fake *Fake) DialReturns(result1 redigo.Conn, result2 error) {
	fake.DialStub = nil
	fake.dialReturns = struct {
		result1 redigo.Conn
		result2 error
	}{result1, result2}
}

func (fake *Fake) DialReturnsOnCall(i int, result1 redigo.Conn, result2 error) {
	fake.DialStub = nil
	if fake.dialReturnsOnCall == nil {
		fake.dialReturnsOnCall = make(map[int]struct {
			result1 redigo.Conn
			result2 error
		})
	}
	fake.dialReturnsOnCall[i] = struct {
		result1 redigo.Conn
		result2 error
	}{result1, result2}
}

func (fake *Fake) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.dialMutex.RLock()
	defer fake.dialMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *Fake) recordInvocation(key string, args []interface{}) {
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

var _ Redis = new(Fake)
