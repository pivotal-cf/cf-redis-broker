package task

import (
	"fmt"

	"github.com/pivotal-cf/cf-redis-broker/recovery"
)

type generic struct {
	msg string
}

func NewGeneric(msg string) recovery.Task {
	return &generic{
		msg: msg,
	}
}

func (g *generic) Run(artifact recovery.Artifact) (recovery.Artifact, error) {
	fmt.Printf("artifact path: %s\n", artifact.Path())
	fmt.Printf("Generic msg: %s\n", g.msg)
	return artifact, nil
}

func (g *generic) Name() string {
	return "generic"
}
