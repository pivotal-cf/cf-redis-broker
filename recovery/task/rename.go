package task

import (
	"fmt"

	"github.com/pivotal-cf/cf-redis-broker/recovery"
)

type rename struct {
	target string
}

func NewRename(target string) recovery.Task {
	return &rename{
		target: target,
	}
}

func (r *rename) Run(a recovery.Artifact) (recovery.Artifact, error) {
	fmt.Printf("Move %s to %s\n", a.Path(), r.target)
	return recovery.NewArtifact(r.target), nil
}

func (r *rename) Name() string {
	return "rename"
}
