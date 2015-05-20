package backup

import (
	"fmt"

	"github.com/pivotal-cf/cf-redis-broker/recovery"
)

type packager struct {
	artifactTarget string
}

func NewPackager(artifactTarget string) *packager {
	return &packager{
		artifactTarget,
	}
}

func (p *packager) Run(a recovery.Artifact) (recovery.Artifact, error) {
	fmt.Printf("Packaging %s\n", a.Path())

	return recovery.NewArtifact(p.artifactTarget), nil
}

func (p *packager) Name() string {
	return "packager"
}
