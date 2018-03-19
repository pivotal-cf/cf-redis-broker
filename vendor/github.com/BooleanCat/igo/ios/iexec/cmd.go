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

//CmdReal is a wrapper around exec.Cmd that implements iexec.Cmd
type CmdReal struct {
	cmd *exec.Cmd
}

//NewCmd creates a struct that behaves like exec.Cmd
func NewCmd(cmd ...*exec.Cmd) *CmdReal {
	if len(cmd) > 0 {
		return &CmdReal{cmd: cmd[0]}
	}
	return &CmdReal{cmd: new(exec.Cmd)}
}

//CombinedOutput is a wrapper around exec.Cmd.CombinedOutput()
func (c *CmdReal) CombinedOutput() ([]byte, error) {
	return c.cmd.CombinedOutput()
}

//Output is a wrapper around exec.Cmd.Output()
func (c *CmdReal) Output() ([]byte, error) {
	return c.cmd.Output()
}

//Run is a wrapper around exec.Cmd.Run()
func (c *CmdReal) Run() error {
	return c.cmd.Run()
}

//Start is a wrapper around exec.Cmd.Start()
func (c *CmdReal) Start() error {
	return c.cmd.Start()
}

//StderrPipe is a wrapper around exec.Cmd.StderrPipe()
func (c *CmdReal) StderrPipe() (io.ReadCloser, error) {
	return c.cmd.StderrPipe()
}

//StdinPipe is a wrapper around exec.Cmd.StdinPipe()
func (c *CmdReal) StdinPipe() (io.WriteCloser, error) {
	return c.cmd.StdinPipe()
}

//StdoutPipe is a wrapper around exec.Cmd.StdoutPipe()
func (c *CmdReal) StdoutPipe() (io.ReadCloser, error) {
	return c.cmd.StdoutPipe()
}

//Wait is a wrapper around exec.Cmd.Wait()
func (c *CmdReal) Wait() error {
	return c.cmd.Wait()
}

//GetPath is a wrapper around getting exec.Cmd.Path
func (c *CmdReal) GetPath() string {
	return c.cmd.Path
}

//SetPath is a wrapper around setting exec.Cmd.Path
func (c *CmdReal) SetPath(path string) {
	c.cmd.Path = path
}

//GetArgs is a wrapper around getting exec.Cmd.Args
func (c *CmdReal) GetArgs() []string {
	return c.cmd.Args
}

//SetArgs is a wrapper around setting exec.Cmd.Args
func (c *CmdReal) SetArgs(args []string) {
	c.cmd.Args = args
}

//GetEnv is a wrapper around getting exec.Cmd.Env
func (c *CmdReal) GetEnv() []string {
	return c.cmd.Env
}

//SetEnv is a wrapper around setting exec.Cmd.Env
func (c *CmdReal) SetEnv(env []string) {
	c.cmd.Env = env
}

//GetDir is a wrapper around getting exec.Cmd.Dir
func (c *CmdReal) GetDir() string {
	return c.cmd.Dir
}

//SetDir is a wrapper around setting exec.Cmd.Dir
func (c *CmdReal) SetDir(dir string) {
	c.cmd.Dir = dir
}

//GetStdin is a wrapper around getting exec.Cmd.Stdin
func (c *CmdReal) GetStdin() io.Reader {
	return c.cmd.Stdin
}

//SetStdin is a wrapper around setting exec.Cmd.Stdin
func (c *CmdReal) SetStdin(stdin io.Reader) {
	c.cmd.Stdin = stdin
}

//GetStdout is a wrapper around getting exec.Cmd.Stdout
func (c *CmdReal) GetStdout() io.Writer {
	return c.cmd.Stdout
}

//SetStdout is a wrapper around setting exec.Cmd.Stdout
func (c *CmdReal) SetStdout(stdout io.Writer) {
	c.cmd.Stdout = stdout
}

//GetStderr is a wrapper around getting exec.Cmd.Stderr
func (c *CmdReal) GetStderr() io.Writer {
	return c.cmd.Stderr
}

//SetStderr is a wrapper around setting exec.Cmd.Stderr
func (c *CmdReal) SetStderr(stderr io.Writer) {
	c.cmd.Stderr = stderr
}

//GetExtraFiles is a wrapper around getting exec.Cmd.ExtraFiles
func (c *CmdReal) GetExtraFiles() []*os.File {
	return c.cmd.ExtraFiles
}

//SetExtraFiles is a wrapper around setting exec.Cmd.ExtraFiles
func (c *CmdReal) SetExtraFiles(files []*os.File) {
	c.cmd.ExtraFiles = files
}

//GetSysProcAttr is a wrapper around getting exec.Cmd.SysProcAttr
func (c *CmdReal) GetSysProcAttr() *syscall.SysProcAttr {
	return c.cmd.SysProcAttr
}

//SetSysProcAttr is a wrapper around setting exec.Cmd.SysProcAttr
func (c *CmdReal) SetSysProcAttr(attr *syscall.SysProcAttr) {
	c.cmd.SysProcAttr = attr
}

//GetProcess is a wrapper around getting exec.Cmd.Process as an ios.Process
func (c *CmdReal) GetProcess() ios.Process {
	return ios.NewProcess(c.cmd.Process)
}

//SetProcess is a wrapper around setting exec.Cmd.Process
func (c *CmdReal) SetProcess(process *os.Process) {
	c.cmd.Process = process
}

//GetProcessState is a wrapper around getting exec.Cmd.ProcessState
func (c *CmdReal) GetProcessState() *os.ProcessState {
	return c.cmd.ProcessState
}

//SetProcessState is a wrapper around setting exec.Cmd.ProcessState
func (c *CmdReal) SetProcessState(state *os.ProcessState) {
	c.cmd.ProcessState = state
}
