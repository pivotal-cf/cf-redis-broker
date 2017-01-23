// This file was generated by counterfeiter
// counterfeiter -o iredis/statuscmd_fake.go --fake-name StatusCmdFake iredis/statuscmd.go StatusCmd

package iredis

import "sync"

//StatusCmdFake ...
type StatusCmdFake struct {
	ResultStub        func() (string, error)
	resultMutex       sync.RWMutex
	resultArgsForCall []struct{}
	resultReturns     struct {
		result1 string
		result2 error
	}
	ErrStub        func() error
	errMutex       sync.RWMutex
	errArgsForCall []struct{}
	errReturns     struct {
		result1 error
	}
	StringStub        func() string
	stringMutex       sync.RWMutex
	stringArgsForCall []struct{}
	stringReturns     struct {
		result1 string
	}
	ValStub        func() string
	valMutex       sync.RWMutex
	valArgsForCall []struct{}
	valReturns     struct {
		result1 string
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

//NewStatusCmdFake is the preferred way to initialise a StatusCmdFake
func NewStatusCmdFake() *StatusCmdFake {
	return new(StatusCmdFake)
}

//Result ...
func (fake *StatusCmdFake) Result() (string, error) {
	fake.resultMutex.Lock()
	fake.resultArgsForCall = append(fake.resultArgsForCall, struct{}{})
	fake.recordInvocation("Result", []interface{}{})
	fake.resultMutex.Unlock()
	if fake.ResultStub != nil {
		return fake.ResultStub()
	}
	return fake.resultReturns.result1, fake.resultReturns.result2
}

//ResultCallCount ...
func (fake *StatusCmdFake) ResultCallCount() int {
	fake.resultMutex.RLock()
	defer fake.resultMutex.RUnlock()
	return len(fake.resultArgsForCall)
}

//ResultReturns ...
func (fake *StatusCmdFake) ResultReturns(result1 string, result2 error) {
	fake.ResultStub = nil
	fake.resultReturns = struct {
		result1 string
		result2 error
	}{result1, result2}
}

//Err ...
func (fake *StatusCmdFake) Err() error {
	fake.errMutex.Lock()
	fake.errArgsForCall = append(fake.errArgsForCall, struct{}{})
	fake.recordInvocation("Err", []interface{}{})
	fake.errMutex.Unlock()
	if fake.ErrStub != nil {
		return fake.ErrStub()
	}
	return fake.errReturns.result1
}

//ErrCallCount ...
func (fake *StatusCmdFake) ErrCallCount() int {
	fake.errMutex.RLock()
	defer fake.errMutex.RUnlock()
	return len(fake.errArgsForCall)
}

//ErrReturns ...
func (fake *StatusCmdFake) ErrReturns(result1 error) {
	fake.ErrStub = nil
	fake.errReturns = struct {
		result1 error
	}{result1}
}

func (fake *StatusCmdFake) String() string {
	fake.stringMutex.Lock()
	fake.stringArgsForCall = append(fake.stringArgsForCall, struct{}{})
	fake.recordInvocation("String", []interface{}{})
	fake.stringMutex.Unlock()
	if fake.StringStub != nil {
		return fake.StringStub()
	}
	return fake.stringReturns.result1
}

//StringCallCount ...
func (fake *StatusCmdFake) StringCallCount() int {
	fake.stringMutex.RLock()
	defer fake.stringMutex.RUnlock()
	return len(fake.stringArgsForCall)
}

//StringReturns ...
func (fake *StatusCmdFake) StringReturns(result1 string) {
	fake.StringStub = nil
	fake.stringReturns = struct {
		result1 string
	}{result1}
}

//Val ...
func (fake *StatusCmdFake) Val() string {
	fake.valMutex.Lock()
	fake.valArgsForCall = append(fake.valArgsForCall, struct{}{})
	fake.recordInvocation("Val", []interface{}{})
	fake.valMutex.Unlock()
	if fake.ValStub != nil {
		return fake.ValStub()
	}
	return fake.valReturns.result1
}

//ValCallCount ...
func (fake *StatusCmdFake) ValCallCount() int {
	fake.valMutex.RLock()
	defer fake.valMutex.RUnlock()
	return len(fake.valArgsForCall)
}

//ValReturns ...
func (fake *StatusCmdFake) ValReturns(result1 string) {
	fake.ValStub = nil
	fake.valReturns = struct {
		result1 string
	}{result1}
}

//Invocations ...
func (fake *StatusCmdFake) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.resultMutex.RLock()
	defer fake.resultMutex.RUnlock()
	fake.errMutex.RLock()
	defer fake.errMutex.RUnlock()
	fake.stringMutex.RLock()
	defer fake.stringMutex.RUnlock()
	fake.valMutex.RLock()
	defer fake.valMutex.RUnlock()
	return fake.invocations
}

func (fake *StatusCmdFake) recordInvocation(key string, args []interface{}) {
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

var _ StatusCmd = new(StatusCmdFake)
