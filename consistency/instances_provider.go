package consistency

import (
	"encoding/json"
	"os"

	"github.com/pivotal-cf/cf-redis-broker/redis"
)

type InstancesProvider interface {
	Instances() ([]redis.Instance, error)
}

type stateFileAvailableInstances struct {
	path string
}

func NewStateFileAvailableInstances(path string) *stateFileAvailableInstances {
	return &stateFileAvailableInstances{path}
}

// Instances reads and returns the available instances from the state file.
func (s *stateFileAvailableInstances) Instances() ([]redis.Instance, error) {
	reader, err := os.Open(s.path)
	if err != nil {
		return nil, err
	}

	state := &struct {
		AvailableInstances []redis.Instance `json:"available_instances"`
	}{}

	if err := json.NewDecoder(reader).Decode(state); err != nil {
		return nil, err
	}

	return state.AvailableInstances, nil
}
