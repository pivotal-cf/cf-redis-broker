package iexec

import (
	"io"
	"os"
	"os/exec"
	"syscall"

	"github.com/BooleanCat/igo/ios"
)

//CmdProvider is a type alias for exec.Command
type CmdProvider func(name string, args ...string) Cmd

//Cmd is an interface around exec.Cmd
type Cmd interface {
	CombinedOutput() ([]byte, error)
	Output() ([]byte, error)
	Run() error
	Start() error
	StderrPipe() (io.ReadCloser, error)
	StdinPipe() (io.WriteCloser, error)
	StdoutPipe() (io.ReadCloser, error)
	Wait() error
	GetPath() string
	SetPath(string)
	GetArgs() []string
	SetArgs([]string)
	GetEnv() []string
	SetEnv([]string)
	GetDir() string
	SetDir(string)
	GetStdin() io.Reader
	SetStdin(io.Reader)
	GetStdout() io.Writer
	SetStdout(io.Writer)
	GetStderr() io.Writer
	SetStderr(io.Writer)
	GetExtraFiles() []*os.File
	SetExtraFiles([]*os.File)
	GetSysProcAttr() *syscall.SysProcAttr
	SetSysProcAttr(*syscall.SysProcAttr)
	GetProcess() ios.Process
	SetProcess(*os.Process)
	GetProcessState() *os.ProcessState
	SetProcessState(*os.ProcessState)
}

//CmdWrap is a wrapper around exec.Cmd that implements iexec.Cmd
type CmdWrap struct {
	cmd *exec.Cmd
}

//CombinedOutput is a wrapper around exec.Cmd.CombinedOutput()
func (c *CmdWrap) CombinedOutput() ([]byte, error) {
	return c.cmd.CombinedOutput()
}

//Output is a wrapper around exec.Cmd.Output()
func (c *CmdWrap) Output() ([]byte, error) {
	return c.cmd.Output()
}

//Run is a wrapper around exec.Cmd.Run()
func (c *CmdWrap) Run() error {
	return c.cmd.Run()
}

//Start is a wrapper around exec.Cmd.Start()
func (c *CmdWrap) Start() error {
	return c.cmd.Start()
}

//StderrPipe is a wrapper around exec.Cmd.StderrPipe()
func (c *CmdWrap) StderrPipe() (io.ReadCloser, error) {
	return c.cmd.StderrPipe()
}

//StdinPipe is a wrapper around exec.Cmd.StdinPipe()
func (c *CmdWrap) StdinPipe() (io.WriteCloser, error) {
	return c.cmd.StdinPipe()
}

//StdoutPipe is a wrapper around exec.Cmd.StdoutPipe()
func (c *CmdWrap) StdoutPipe() (io.ReadCloser, error) {
	return c.cmd.StdoutPipe()
}

//Wait is a wrapper around exec.Cmd.Wait()
func (c *CmdWrap) Wait() error {
	return c.cmd.Wait()
}

//GetPath is a wrapper around getting exec.Cmd.Path
func (c *CmdWrap) GetPath() string {
	return c.cmd.Path
}

//SetPath is a wrapper around setting exec.Cmd.Path
func (c *CmdWrap) SetPath(path string) {
	c.cmd.Path = path
}

//GetArgs is a wrapper around getting exec.Cmd.Args
func (c *CmdWrap) GetArgs() []string {
	return c.cmd.Args
}

//SetArgs is a wrapper around setting exec.Cmd.Args
func (c *CmdWrap) SetArgs(args []string) {
	c.cmd.Args = args
}

//GetEnv is a wrapper around getting exec.Cmd.Env
func (c *CmdWrap) GetEnv() []string {
	return c.cmd.Env
}

//SetEnv is a wrapper around setting exec.Cmd.Env
func (c *CmdWrap) SetEnv(env []string) {
	c.cmd.Env = env
}

//GetDir is a wrapper around getting exec.Cmd.Dir
func (c *CmdWrap) GetDir() string {
	return c.cmd.Dir
}

//SetDir is a wrapper around setting exec.Cmd.Dir
func (c *CmdWrap) SetDir(dir string) {
	c.cmd.Dir = dir
}

//GetStdin is a wrapper around getting exec.Cmd.Stdin
func (c *CmdWrap) GetStdin() io.Reader {
	return c.cmd.Stdin
}

//SetStdin is a wrapper around setting exec.Cmd.Stdin
func (c *CmdWrap) SetStdin(stdin io.Reader) {
	c.cmd.Stdin = stdin
}

//GetStdout is a wrapper around getting exec.Cmd.Stdout
func (c *CmdWrap) GetStdout() io.Writer {
	return c.cmd.Stdout
}

//SetStdout is a wrapper around setting exec.Cmd.Stdout
func (c *CmdWrap) SetStdout(stdout io.Writer) {
	c.cmd.Stdout = stdout
}

//GetStderr is a wrapper around getting exec.Cmd.Stderr
func (c *CmdWrap) GetStderr() io.Writer {
	return c.cmd.Stderr
}

//SetStderr is a wrapper around setting exec.Cmd.Stderr
func (c *CmdWrap) SetStderr(stderr io.Writer) {
	c.cmd.Stderr = stderr
}

//GetExtraFiles is a wrapper around getting exec.Cmd.ExtraFiles
func (c *CmdWrap) GetExtraFiles() []*os.File {
	return c.cmd.ExtraFiles
}

//SetExtraFiles is a wrapper around setting exec.Cmd.ExtraFiles
func (c *CmdWrap) SetExtraFiles(files []*os.File) {
	c.cmd.ExtraFiles = files
}

//GetSysProcAttr is a wrapper around getting exec.Cmd.SysProcAttr
func (c *CmdWrap) GetSysProcAttr() *syscall.SysProcAttr {
	return c.cmd.SysProcAttr
}

//SetSysProcAttr is a wrapper around setting exec.Cmd.SysProcAttr
func (c *CmdWrap) SetSysProcAttr(attr *syscall.SysProcAttr) {
	c.cmd.SysProcAttr = attr
}

//GetProcess is a wrapper around getting exec.Cmd.Process as an ios.Process
func (c *CmdWrap) GetProcess() ios.Process {
	return ios.NewProcessWrap(c.cmd.Process)
}

//SetProcess is a wrapper around setting exec.Cmd.Process
func (c *CmdWrap) SetProcess(process *os.Process) {
	c.cmd.Process = process
}

//GetProcessState is a wrapper around getting exec.Cmd.ProcessState
func (c *CmdWrap) GetProcessState() *os.ProcessState {
	return c.cmd.ProcessState
}

//SetProcessState is a wrapper around setting exec.Cmd.ProcessState
func (c *CmdWrap) SetProcessState(state *os.ProcessState) {
	c.cmd.ProcessState = state
}
