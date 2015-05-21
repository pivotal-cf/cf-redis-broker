package task

type Artifact interface {
	Path() string
}

type artifact struct {
	path string
}

func NewArtifact(path string) Artifact {
	return &artifact{path}
}

func (a *artifact) Path() string {
	return a.path
}

type Task interface {
	Name() string
	Run(Artifact) (Artifact, error)
}
