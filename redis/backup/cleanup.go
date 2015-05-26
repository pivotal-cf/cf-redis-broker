package backup

import (
	"os"

	"github.com/pivotal-cf/cf-redis-broker/recovery/task"
)

type cleanup struct {
	dumpRdbFile    string
	renamedRdbFile string
}

func NewCleanup(dumpRdbFile, renamedRdbFile string) task.Task {
	return &cleanup{
		dumpRdbFile:    dumpRdbFile,
		renamedRdbFile: renamedRdbFile,
	}
}

func (c *cleanup) Run(artifact task.Artifact) (task.Artifact, error) {
	if !fileExists(c.dumpRdbFile) {
		if err := os.Rename(c.renamedRdbFile, c.dumpRdbFile); err != nil {
			return artifact, err
		}
	}
	if fileExists(c.renamedRdbFile) {
		if err := os.Remove(c.renamedRdbFile); err != nil {
			return artifact, err
		}
	}
	return artifact, nil
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

func (c *cleanup) Name() string {
	return "cleanup"
}
