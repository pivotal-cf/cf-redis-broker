package backup

import (
	"fmt"

	"github.com/pivotal-cf/cf-redis-broker/recovery/task"
)

type cleanup struct {
	target  string
	source  string
	archive string
}

func NewCleanup(target, source, archive string) task.Task {
	return &cleanup{
		target:  target,
		source:  source,
		archive: archive,
	}
}

func (c *cleanup) Run(artifact task.Artifact) (task.Artifact, error) {
	fmt.Printf("Check if %s exists\n", c.target)
	fmt.Printf("Move/delete %s\n", c.source)
	fmt.Printf("delete %s\n", c.archive)
	return artifact, nil
}

func (c *cleanup) Name() string {
	return "cleanup"
}
