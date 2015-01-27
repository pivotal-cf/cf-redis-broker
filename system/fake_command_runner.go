package system

import (
	"strings"
)

type FakeCommandRunner struct {
	Commands []string

	RunError error
}

func (fakeCommandRunner *FakeCommandRunner) Run(name string, args ...string) error {
	if fakeCommandRunner.RunError != nil {
		return fakeCommandRunner.RunError
	}

	command := name + " " + strings.Join(args, " ")
	fakeCommandRunner.Commands = append(fakeCommandRunner.Commands, command)
	return nil
}
