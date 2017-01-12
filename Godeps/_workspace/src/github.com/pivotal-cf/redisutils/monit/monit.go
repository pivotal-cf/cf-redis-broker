package monit

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/BooleanCat/igo/ios/iexec"
)

//ErrTimeout indicates that some monit action took too long
var ErrTimeout = errors.New("timed out waiting for monit operation")

//Status is an enumeration of monit statuses
type Status int

//Statuses is a mapping of monit statuses to Status
type Statuses map[string]Status

const (
	_ Status = iota

	//StatusRunning indicates `Process 'foo' running`
	StatusRunning

	//StatusNotMonitored indicates `Process 'foo' not monitored`
	StatusNotMonitored

	//StatusNotMonitoredStartPending indicates `Process 'foo' not monitored - start pending`
	StatusNotMonitoredStartPending

	//StatusInitializing indicates `Process 'foo' initializing`
	StatusInitializing

	//StatusDoesNotExist indicates `Process 'foo' Does not exist`
	StatusDoesNotExist

	//StatusNotMonitoredStopPending indicates `Process 'foo' not monitored - stop pending`
	StatusNotMonitoredStopPending

	//StatusRunningRestartPending indicates `Process 'foo' running - restart pending`
	StatusRunningRestartPending
)

var statusMapping = Statuses{
	"running":                       StatusRunning,
	"not monitored":                 StatusNotMonitored,
	"not monitored - start pending": StatusNotMonitoredStartPending,
	"initializing":                  StatusInitializing,
	"Does not exist":                StatusDoesNotExist,
	"not monitored - stop pending":  StatusNotMonitoredStopPending,
	"running - restart pending":     StatusRunningRestartPending,
}

func getStatus(status string) Status {
	return statusMapping[status]
}

//Monit is a controller for the monit CLI
type Monit interface {
	GetSummary() (Statuses, error)
	GetStatus(string) (Status, error)
	Start(string) error
	Stop(string) error
	StartAndWait(string) error
	StopAndWait(string) error
	SetMonitrcPath(string)
	SetExecutable(string)
}

//SysMonit is a controller for the monit CLI
type SysMonit struct {
	MonitrcPath string
	Executable  string

	interval time.Duration
	timeout  time.Duration
	exec     iexec.Exec
}

//New is the correct way to initialise a new Monit
func New() *SysMonit {
	return &SysMonit{
		Executable: "monit",
		interval:   time.Millisecond * 100,
		timeout:    time.Second * 15,
		exec:       iexec.New(),
	}
}

//SetExecutable updates monit's executable path
func (monit *SysMonit) SetExecutable(path string) {
	monit.Executable = path
}

//SetMonitrcPath updates monit's target monitrc when making CLI calls
func (monit *SysMonit) SetMonitrcPath(path string) {
	monit.MonitrcPath = path
}

//GetSummary is synonymous with `monit summary`
func (monit *SysMonit) GetSummary() (Statuses, error) {
	rawSummary, err := monit.getRawSummary()
	if err != nil {
		return nil, err
	}

	processes := monit.getProcessesFromRawSummary(rawSummary)
	return monit.newProcessMap(processes), nil
}

//GetStatus a job specific Status from GetSummary
func (monit *SysMonit) GetStatus(job string) (Status, error) {
	summary, err := monit.GetSummary()
	if err != nil {
		return 0, err
	}

	status, ok := summary[job]
	if !ok {
		return status, fmt.Errorf("no such job: `%s`", job)
	}
	return status, nil
}

//Start is synonymous with `monit start {job}`
func (monit *SysMonit) Start(job string) error {
	cmd := monit.getMonitCommand("start", job)
	return monit.setErrorContentOf(cmd.CombinedOutput())
}

//Stop is synonymous with `monit stop {job}`
func (monit *SysMonit) Stop(job string) error {
	cmd := monit.getMonitCommand("stop", job)
	return monit.setErrorContentOf(cmd.CombinedOutput())
}

//StartAndWait runs Start(job) and waits for GetStatus(job) to report StatusRunning
func (monit *SysMonit) StartAndWait(job string) error {
	err := monit.Start(job)
	if err != nil {
		return err
	}

	return monit.waitFor(job, StatusRunning)
}

//StopAndWait runs Stop(job) and waits for GetStatus(job) to report StatusNotMonitored
func (monit *SysMonit) StopAndWait(job string) error {
	err := monit.Stop(job)
	if err != nil {
		return err
	}

	return monit.waitFor(job, StatusNotMonitored)
}

func (monit *SysMonit) waitFor(job string, status Status) error {
	for elapsed := time.Duration(0); elapsed < monit.timeout; elapsed = elapsed + monit.interval {
		done, doneErr := monit.jobHasStatus(job, status)

		if doneErr != nil || done {
			return doneErr
		}

		time.Sleep(monit.interval)
	}

	return ErrTimeout
}

func (monit *SysMonit) jobHasStatus(job string, status Status) (bool, error) {
	if job == "all" {
		return monit.allJobsHaveStatus(status)
	}

	currentStatus, err := monit.GetStatus(job)
	return status == currentStatus, err
}

func (monit *SysMonit) setErrorContentOf(output []byte, err error) error {
	if err != nil {
		err = errors.New(string(output))
	}
	return err
}

func (monit *SysMonit) allJobsHaveStatus(status Status) (bool, error) {
	summary, err := monit.GetSummary()
	if err != nil {
		return false, err
	}

	for _, jobStatus := range summary {
		if jobStatus != status {
			return false, nil
		}
	}

	return true, nil
}

func (monit *SysMonit) getRawSummary() (string, error) {
	cmd := monit.getMonitCommand("summary")

	rawSummary, err := cmd.CombinedOutput()
	return string(rawSummary), err
}

func (monit *SysMonit) getProcessesFromRawSummary(summary string) [][]string {
	pattern := regexp.MustCompile(`(?m)^Process '([\w\-]+)'\s+([\w \-]+)$`)
	return pattern.FindAllStringSubmatch(summary, -1)
}

func (monit *SysMonit) newProcessMap(processes [][]string) Statuses {
	processMap := make(Statuses)
	for _, process := range processes {
		processMap[process[1]] = getStatus(process[2])
	}

	return processMap
}

func (monit *SysMonit) getMonitCommand(args ...string) iexec.Cmd {
	var allArgs []string

	if monit.MonitrcPath != "" {
		allArgs = []string{"-c", monit.MonitrcPath}
	}

	allArgs = append(allArgs, args...)
	return monit.exec.Command(monit.Executable, allArgs...)
}
