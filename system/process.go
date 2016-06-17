package system

import "github.com/shirou/gopsutil/process"

type ProcessInfo interface {
	Name(pid int) (string, error)
}

type OSProcessInfo struct{}

func (processInfo *OSProcessInfo) Name(pid int) (string, error) {
	process, err := process.NewProcess(int32(pid))
	if err != nil {
		return "", err
	}

	return process.Name()
}
