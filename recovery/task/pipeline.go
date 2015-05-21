package task

import "github.com/pivotal-golang/lager"

type pipeline struct {
	name   string
	logger lager.Logger
	tasks  []Task
}

func NewPipeline(
	name string,
	logger lager.Logger,
	tasks ...Task,
) Task {
	return &pipeline{
		logger: logger,
		tasks:  tasks,
		name:   name,
	}
}

func (p *pipeline) Name() string {
	return p.name
}

func (p *pipeline) Run(artifact Artifact) (Artifact, error) {
	var err error
	for _, task := range p.tasks {
		p.logInfo("starting", task)

		artifact, err = task.Run(artifact)
		if err != nil {
			p.logError(err, task)
			return nil, err
		}

		p.logInfo("done", task)
	}
	return artifact, err
}

func (p *pipeline) logInfo(event string, task Task) {
	p.logger.Info("pipleline-step",
		lager.Data{
			"event":    event,
			"pipeline": p.Name(),
			"task":     task.Name(),
		},
	)
}

func (p *pipeline) logError(err error, task Task) {
	p.logger.Error("pipleline-step",
		err,
		lager.Data{
			"event":    "failed",
			"pipeline": p.Name(),
			"task":     task.Name(),
		},
	)
}
