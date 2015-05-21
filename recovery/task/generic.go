package task

import "fmt"

type generic struct {
	msg string
}

func NewGeneric(msg string) Task {
	return &generic{
		msg: msg,
	}
}

func (g *generic) Run(artifact Artifact) (Artifact, error) {
	fmt.Printf("artifact path: %s\n", artifact.Path())
	fmt.Printf("Generic msg: %s\n", g.msg)
	return artifact, nil
}

func (g *generic) Name() string {
	return "generic"
}
