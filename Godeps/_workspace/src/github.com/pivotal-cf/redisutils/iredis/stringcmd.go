package iredis

import redis "gopkg.in/redis.v5"

//StringCmd is an interface around redis.StringCmd
type StringCmd interface {
	Result() (string, error)
	Err() error
	String() string
	Val() string
}

//StringCmdReal is a wrapper around redis that implements iredis.StringCmd
type StringCmdReal struct {
	stringCmd *redis.StringCmd
}

//Result is a wrapper around redis.StringCmd.Result()
func (cmd *StringCmdReal) Result() (string, error) {
	return cmd.stringCmd.Result()
}

//Err is a wrapper around redis.StringCmd.Err()
func (cmd *StringCmdReal) Err() error {
	return cmd.stringCmd.Err()
}

//String is a wrapper around redis.StringCmd.String()
func (cmd *StringCmdReal) String() string {
	return cmd.stringCmd.String()
}

//Val is a wrapper around redis.StringCmd.Val()
func (cmd *StringCmdReal) Val() string {
	return cmd.stringCmd.Val()
}
