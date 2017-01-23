package iredis

import redis "gopkg.in/redis.v5"

//StatusCmd is an interface around redis.StatusCmd
type StatusCmd interface {
	Result() (string, error)
	Err() error
	String() string
	Val() string
}

//StatusCmdReal is a wrapper around redis that implements iredis.StatusCmd
type StatusCmdReal struct {
	statusCmd *redis.StatusCmd
}

//NewStatusCmd is a wrapper around redis.NewStatusCmd()
func NewStatusCmd(args ...interface{}) StatusCmd {
	return &StatusCmdReal{statusCmd: redis.NewStatusCmd(args...)}
}

//Result is a wrapper around redis.StatusCmd.Result()
func (cmd *StatusCmdReal) Result() (string, error) {
	return cmd.statusCmd.Result()
}

//Err is a wrapper around redis.StatusCmd.Err()
func (cmd *StatusCmdReal) Err() error {
	return cmd.statusCmd.Err()
}

//Val is a wrapper around redis.StatusCmd.Val()
func (cmd *StatusCmdReal) Val() string {
	return cmd.statusCmd.Val()
}

//String is a wrapper around redis.StatusCmd.String()
func (cmd *StatusCmdReal) String() string {
	return cmd.statusCmd.String()
}
