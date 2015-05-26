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
	if _, err := os.Stat(c.dumpRdbFile); os.IsNotExist(err) {
		os.Rename(c.renamedRdbFile, c.dumpRdbFile)
	} else if err != nil {
		return nil, err
	} else {
		err = os.Remove(c.renamedRdbFile)
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
	}
	return artifact, nil
}

// func fileExists(filename string) {
// }

func (c *cleanup) Name() string {
	return "cleanup"
}
