package task

import "fmt"

type rename struct {
	target string
}

func NewRename(target string) Task {
	return &rename{
		target: target,
	}
}

func (r *rename) Run(a Artifact) (Artifact, error) {
	fmt.Printf("Move %s to %s\n", a.Path(), r.target)
	return NewArtifact(r.target), nil
}

func (r *rename) Name() string {
	return "rename"
}
