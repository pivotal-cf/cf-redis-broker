package iexec

import (
	"bytes"
	"io/ioutil"
	"os/exec"

	"github.com/BooleanCat/igo/ios"
)

//Exec is an interface around os/exec
type Exec interface {
	Command(string, ...string) Cmd
}

/*
PureFake returns a struct containing fake Exec with nested initialised fake
members.

The following Fakes are available:
- Cmd: a FakeCmd returned by Exec.Command()
- Process: a FakeProcess returned by Cmd.GetProcess()
*/
type PureFake struct {
	Exec    *ExecFake
	Cmd     *CmdFake
	Process *ios.ProcessFake
}

//NewPureFake returns a fake Exec with nested new fakes within
func NewPureFake() *PureFake {
	execFake := new(ExecFake)
	processFake := new(ios.ProcessFake)
	cmdFake := newPureCmdFake(processFake)
	execFake.CommandReturns(cmdFake)
	return &PureFake{
		Exec:    execFake,
		Cmd:     cmdFake,
		Process: processFake,
	}
}

func newPureCmdFake(process ios.Process) *CmdFake {
	fake := new(CmdFake)
	fake.GetProcessReturns(process)
	fake.StdoutPipeReturns(ioutil.NopCloser(new(bytes.Buffer)), nil)
	fake.StderrPipeReturns(ioutil.NopCloser(new(bytes.Buffer)), nil)
	return fake
}

//ExecWrap is a wrapper around exec that implements iexec.Exec
type ExecWrap struct{}

//Command is a wrapper around exec.Command()
func (e *ExecWrap) Command(name string, args ...string) Cmd {
	return &CmdWrap{cmd: exec.Command(name, args...)}
}
