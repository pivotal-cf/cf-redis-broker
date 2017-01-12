package iexec

import "os/exec"

//Exec is an interface around os/exec
type Exec interface {
	Command(string, ...string) Cmd
}

//Real is a wrapper around exec that implements iexec.Exec
type Real struct{}

//New creates a struct that behaves like the exec package
func New() *Real {
	return new(Real)
}

//Command is a wrapper around exec.Command()
func (*Real) Command(name string, args ...string) Cmd {
	return NewCmd(exec.Command(name, args...))
}
