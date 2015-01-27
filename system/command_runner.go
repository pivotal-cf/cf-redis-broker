package system

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/pivotal-golang/lager"
)

type CommandRunner interface {
	Run(name string, args ...string) error
}

type OSCommandRunner struct {
	Logger lager.Logger
}

func (runner OSCommandRunner) Run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	runner.Logger.Debug(fmt.Sprint(name, " ", strings.Join(args, " ")))
	output, err := cmd.CombinedOutput()
	if err != nil {
		runner.Logger.Info(fmt.Sprintf("command failed with output: %s", output))
	}
	return err
}
