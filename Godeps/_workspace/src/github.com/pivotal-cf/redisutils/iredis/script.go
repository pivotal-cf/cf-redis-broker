package iredis

import redis "gopkg.in/redis.v5"

//Scripter ...
type Scripter interface {
	Eval(script string, keys []string, args ...interface{}) *redis.Cmd
	EvalSha(sha1 string, keys []string, args ...interface{}) *redis.Cmd
	ScriptExists(scripts ...string) *redis.BoolSliceCmd
	ScriptLoad(script string) *redis.StringCmd
}

//Script is an interface around redis.Script
type Script interface {
	Eval(Scripter, []string, ...interface{}) *redis.Cmd
	EvalSha(Scripter, []string, ...interface{}) *redis.Cmd
	Exists(Scripter) BoolSliceCmd
	Load(Scripter) StringCmd
	Run(Scripter, []string, ...interface{}) *redis.Cmd
}

//ScriptReal is a wrapper around redis that implements iredis.Script
type ScriptReal struct {
	script *redis.Script
}

//NewScript is a wrapper around redis.NewScript()
func (*Real) NewScript(src string) Script {
	return &ScriptReal{script: redis.NewScript(src)}
}

//Eval is a wrapper around redis.Script.Eval()
func (s *ScriptReal) Eval(c Scripter, keys []string, args ...interface{}) *redis.Cmd {
	return s.script.Eval(c, keys, args...)
}

//EvalSha is a wrapper around redis.Script.EvalSha()
func (s *ScriptReal) EvalSha(c Scripter, keys []string, args ...interface{}) *redis.Cmd {
	return s.script.EvalSha(c, keys, args...)
}

//Exists is a wrapper around redis.Script.Exists()
func (s *ScriptReal) Exists(c Scripter) BoolSliceCmd {
	return &BoolSliceCmdReal{boolSliceCmd: s.script.Exists(c)}
}

//Load is a wrapper around redis.Script.Load()
func (s *ScriptReal) Load(c Scripter) StringCmd {
	return &StringCmdReal{stringCmd: s.script.Load(c)}
}

//Run is a wrapper around redis.Script.Run()
func (s *ScriptReal) Run(c Scripter, keys []string, args ...interface{}) *redis.Cmd {
	return s.script.Run(c, keys, args...)
}
