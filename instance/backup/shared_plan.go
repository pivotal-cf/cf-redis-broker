package backup

import (
	"errors"
	"path/filepath"
)

type sharedPlan struct{}

func (p *sharedPlan) InstanceID(redisConfigPath, nodeIP string) (string, error) {
	cleanPath := filepath.Clean(redisConfigPath)

	dir, _ := filepath.Split(cleanPath)
	if dir == "" {
		return "", errors.New("Invalid config path")
	}

	instanceID := filepath.Base(dir)
	return instanceID, nil
}
