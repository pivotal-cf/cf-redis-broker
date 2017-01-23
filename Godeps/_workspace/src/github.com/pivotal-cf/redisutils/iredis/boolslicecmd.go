package iredis

import redis "gopkg.in/redis.v5"

//BoolSliceCmd is an interface around redis.BoolSliceCmd
type BoolSliceCmd interface {
	Result() ([]bool, error)
	Err() error
	String() string
	Val() []bool
}

//BoolSliceCmdReal is a wrapper around redis that implements iredis.BoolSliceCmd
type BoolSliceCmdReal struct {
	boolSliceCmd *redis.BoolSliceCmd
}

//NewBoolSliceCmd is a wrapper around redis.NewBoolSliceCmd()
func NewBoolSliceCmd(args ...interface{}) BoolSliceCmd {
	return &BoolSliceCmdReal{boolSliceCmd: redis.NewBoolSliceCmd(args...)}
}

//Result is a wrapper around redis.BoolSliceCmd.Result()
func (bsc *BoolSliceCmdReal) Result() ([]bool, error) {
	return bsc.boolSliceCmd.Result()
}

//Err is a wrapper around redis.BoolSliceCmd.Err()
func (bsc *BoolSliceCmdReal) Err() error {
	return bsc.boolSliceCmd.Err()
}

//String is a wrapper around redis.BoolSliceCmd.String()
func (bsc *BoolSliceCmdReal) String() string {
	return bsc.boolSliceCmd.String()
}

//Val is a wrapper around redis.BoolSliceCmd.Val()
func (bsc *BoolSliceCmdReal) Val() []bool {
	return bsc.boolSliceCmd.Val()
}
