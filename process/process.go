package process

import (
	"os"
	"syscall"
)

type ProcessChecker struct{}
type ProcessKiller struct{}

func (*ProcessChecker) Alive(pid int) bool {
	osProcess, findProcessErr := os.FindProcess(pid)
	if findProcessErr != nil {
		return false
	}
	err := osProcess.Signal(syscall.Signal(0))
	if err != nil {
		return false
	}

	return true
}

func (*ProcessKiller) Kill(pid int) error {
	osProcess, findProcessErr := os.FindProcess(pid)
	if findProcessErr != nil {
		return findProcessErr
	}

	killProcessErr := osProcess.Kill()
	if killProcessErr != nil {
		return killProcessErr
	}

	osProcess.Wait()
	return nil
}
