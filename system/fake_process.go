package system

type FakeOSProcessInfo struct {
	NameReturns string
	NameErr     error
}

func (processInfo *FakeOSProcessInfo) Name(pid int) (string, error) {
	return processInfo.NameReturns, processInfo.NameErr
}
