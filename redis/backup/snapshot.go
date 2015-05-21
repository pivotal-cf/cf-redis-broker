package backup

import (
	"github.com/pivotal-cf/cf-redis-broker/recovery"
	"github.com/pivotal-cf/cf-redis-broker/recovery/task"
	redis "github.com/pivotal-cf/cf-redis-broker/redis/client"
	"github.com/pivotal-golang/lager"
)

type snapshot struct {
	client  redis.Client
	timeout int
	logger  lager.Logger
}

func NewSnapshot(client redis.Client, timeout int, logger lager.Logger) recovery.Snapshot {
	return &snapshot{
		client:  client,
		timeout: timeout,
		logger:  logger,
	}
}

func (s *snapshot) Create() (task.Artifact, error) {
	s.logger.Info("snapshot",
		lager.Data{
			"task":    "create-snapshot",
			"event":   "starting",
			"timeout": s.timeout,
		},
	)

	if err := s.client.CreateSnapshot(s.timeout); err != nil {
		s.logger.Error("snapshot",
			err,
			lager.Data{
				"task":  "create-snapshot",
				"event": "failed",
			},
		)
		return nil, err
	}

	s.logger.Info("snapshot",
		lager.Data{
			"task":    "create-snapshot",
			"event":   "done",
			"timeout": s.timeout,
		},
	)

	s.logger.Info("snapshot",
		lager.Data{
			"task":  "get-rdb-path",
			"event": "starting",
		},
	)

	path, err := s.client.RDBPath()
	if err != nil {
		s.logger.Error("snapshot",
			err,
			lager.Data{
				"task":  "get-rdb-path",
				"event": "failed",
			},
		)
		return nil, err
	}

	s.logger.Info("snapshot",
		lager.Data{
			"task":  "get-rdb-path",
			"event": "done",
			"path":  path,
		},
	)

	return task.NewArtifact(path), nil
}
