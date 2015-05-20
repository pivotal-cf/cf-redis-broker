package backup

import (
	"github.com/pivotal-cf/cf-redis-broker/recovery"
	redis "github.com/pivotal-cf/cf-redis-broker/redis/client"
)

type Snapshot struct {
	Client redis.Client
}

func (s *Snapshot) Create() (recovery.Artifact, error) {
	// connect to redis
	// run a bgsave
	// wait for bgsave to finish
	// check bgsave status
	// build artifact object with filepath
	return recovery.NewArtifact("/redis/dump.rdb"), nil
}
