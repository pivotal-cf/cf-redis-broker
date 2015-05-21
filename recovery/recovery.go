package recovery

import "github.com/pivotal-cf/cf-redis-broker/recovery/task"

type Snapshot interface {
	Create() (task.Artifact, error)
}
