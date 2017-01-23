package iredis

import redis "gopkg.in/redis.v5"

//Client is an interface around redis.Client
type Client interface {
	Close() error
	Ping() StatusCmd
	BgRewriteAOF() StatusCmd
	Info(...string) StringCmd
	ScriptKill() StatusCmd
}

//ClientReal is a wrapper around redis that implements iredis.Client
type ClientReal struct {
	client *redis.Client
}

//NewClient is a wrapper around redis.NewClient()
func (*Real) NewClient(opt *redis.Options) Client {
	return &ClientReal{client: redis.NewClient(opt)}
}

//Close is a wrapper around redis.Client.Close()
func (c *ClientReal) Close() error {
	return c.client.Close()
}

//Ping is a wrapper around redis.Client.Ping()
func (c *ClientReal) Ping() StatusCmd {
	return &StatusCmdReal{statusCmd: c.client.Ping()}
}

//BgRewriteAOF is a wrapper around redis.Client.BgRewriteAOF()
func (c *ClientReal) BgRewriteAOF() StatusCmd {
	return &StatusCmdReal{statusCmd: c.client.BgRewriteAOF()}
}

//Info is a wrapper around redis.Client.Info()
func (c *ClientReal) Info(section ...string) StringCmd {
	return &StringCmdReal{stringCmd: c.client.Info(section...)}
}

//ScriptKill is a wrapper around redis.Client.ScriptKill()
func (c *ClientReal) ScriptKill() StatusCmd {
	return &StatusCmdReal{statusCmd: c.client.ScriptKill()}
}
