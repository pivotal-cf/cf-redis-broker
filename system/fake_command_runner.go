package system

import (
	"strings"
)

type FakeCommandRunner struct {
	Commands []string

	RunError              error
	CombinedOutputError   error
	CombinedOutputReturns []byte
}

func (fakeCommandRunner *FakeCommandRunner) Run(name string, args ...string) error {
	if fakeCommandRunner.RunError != nil {
		return fakeCommandRunner.RunError
	}

	command := name + " " + strings.Join(args, " ")
	fakeCommandRunner.Commands = append(fakeCommandRunner.Commands, command)
	return nil
}

func (runner *FakeCommandRunner) CombinedOutput(name string, args ...string) ([]byte, error) {
	if runner.CombinedOutputError != nil {
		return runner.CombinedOutputReturns, runner.CombinedOutputError
	}

	command := name + " " + strings.Join(args, " ")
	runner.Commands = append(runner.Commands, command)
	return runner.CombinedOutputReturns, nil
}
