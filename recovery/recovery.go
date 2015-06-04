package recovery

import "github.com/pivotal-cf/cf-redis-broker/recovery/task"

type Snapshotter interface {
	Snapshot() (task.Artifact, error)
}
