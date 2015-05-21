package task

import "github.com/pivotal-golang/lager"

type Pipeline struct {
	name   string
	logger lager.Logger
	tasks  []Task
}

func NewPipeline(
	name string,
	logger lager.Logger,
	tasks ...Task,
) Task {
	return &Pipeline{
		logger: logger,
		tasks:  tasks,
		name:   name,
	}
}

func (p *Pipeline) Name() string {
	return p.name
}

func (p *Pipeline) Run(artifact Artifact) (Artifact, error) {
	var err error
	for _, task := range p.tasks {
		p.logger.Info("pipleline-task",
			lager.Data{
				"event":    "starting",
				"pipeline": p.Name(),
				"task":     task.Name(),
			},
		)

		artifact, err = task.Run(artifact)
		if err != nil {
			p.logger.Error("pipleline-step",
				err,
				lager.Data{
					"event": "failed",
				},
			)
			return nil, err
		}

		p.logger.Info("pipleline-step",
			lager.Data{
				"event": "done",
			},
		)
	}
	return artifact, err
}
