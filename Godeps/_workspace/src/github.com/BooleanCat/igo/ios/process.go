package ios

import "os"

//Process is an interface around os.Process
type Process interface {
	Kill() error
	Release() error
	Signal(os.Signal) error
	Wait() (*os.ProcessState, error)
	GetPid() int
	SetPid(int)
}

//NewProcessWrap creates a ProcessWrap from a os.Process
func NewProcessWrap(process *os.Process) Process {
	return &ProcessWrap{process: process}
}

//ProcessWrap is a wrapper around os.Process that implements ios.Process
type ProcessWrap struct {
	process *os.Process
}

//Kill is a wrapper around os.Process.Kill()
func (p *ProcessWrap) Kill() error {
	return p.process.Kill()
}

//Release is a wrapper around os.Process.Release()
func (p *ProcessWrap) Release() error {
	return p.process.Release()
}

//Signal is a wrapper around os.Process.Signal()
func (p *ProcessWrap) Signal(sig os.Signal) error {
	return p.process.Signal(sig)
}

//Wait is a wrapper around os.Process.Wait()
func (p *ProcessWrap) Wait() (*os.ProcessState, error) {
	return p.process.Wait()
}

//GetPid is a wrapper around getting os.Process.Pid
func (p *ProcessWrap) GetPid() int {
	return p.process.Pid
}

//SetPid is a wrapper around setting os.Process.Pid
func (p *ProcessWrap) SetPid(pid int) {
	p.process.Pid = pid
}
