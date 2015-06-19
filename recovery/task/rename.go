package task

import (
	"github.com/pivotal-cf/cf-redis-broker/utils"
	"github.com/pivotal-golang/lager"
)

type rename struct {
	target string
	logger lager.Logger
}

func NewRename(target string, logger lager.Logger) Task {
	return &rename{
		target: target,
		logger: logger,
	}
}

func (r *rename) Run(a Artifact) (Artifact, error) {
	r.logInfo("starting", a.Path())

	if err := utils.MoveFile(a.Path(), r.target); err != nil {
		r.logError(err, a.Path())
		return nil, err
	}

	r.logInfo("done", a.Path())

	return NewArtifact(r.target), nil
}

func (r *rename) Name() string {
	return "rename"
}

func (r *rename) logInfo(event string, source string) {
	r.logger.Info(r.Name(),
		lager.Data{
			"event":  event,
			"source": source,
			"target": r.target,
			"task":   r.Name(),
		},
	)
}

func (r *rename) logError(err error, source string) {
	r.logger.Error(r.Name(),
		err,
		lager.Data{
			"event":  "failed",
			"source": source,
			"target": r.target,
			"task":   r.Name(),
		},
	)
}
