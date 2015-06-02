package shared

import (
	"errors"
	"path/filepath"

	"github.com/pivotal-cf/cf-redis-broker/instance"
)

type idProvider struct{}

func InstanceIDProvider() instance.IDProvider {
	return &idProvider{}
}

func (p *idProvider) InstanceID(redisConfigPath, nodeIP string) (string, error) {
	cleanPath := filepath.Clean(redisConfigPath)

	dir, _ := filepath.Split(cleanPath)
	if dir == "" {
		return "", errors.New("Invalid config path")
	}

	instanceID := filepath.Base(dir)
	return instanceID, nil
}
