package backup

import (
	"fmt"

	"github.com/pivotal-cf/cf-redis-broker/recovery/task"
)

type packager struct {
	artifactTarget string
}

func NewPackager(artifactTarget string) *packager {
	return &packager{
		artifactTarget,
	}
}

func (p *packager) Run(a task.Artifact) (task.Artifact, error) {
	fmt.Printf("Packaging %s\n", a.Path())

	return task.NewArtifact(p.artifactTarget), nil
}

func (p *packager) Name() string {
	return "packager"
}
