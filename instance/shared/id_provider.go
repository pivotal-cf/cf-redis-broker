package shared

import (
	"errors"
	"path/filepath"

	"github.com/pivotal-cf/cf-redis-broker/instance"
	"github.com/pivotal-golang/lager"
)

type idProvider struct {
	logger lager.Logger
}

func InstanceIDProvider(logger lager.Logger) instance.IDProvider {
	return &idProvider{
		logger: logger,
	}
}

func (p *idProvider) InstanceID(redisConfigPath, nodeIP string) (string, error) {
	p.logger.Info("shared-instance-id", lager.Data{
		"event": "starting",
		"path":  redisConfigPath,
	})

	cleanPath := filepath.Clean(redisConfigPath)

	dir, _ := filepath.Split(cleanPath)
	if dir == "" {
		err := errors.New("Invalid config path")
		p.logger.Error("shared-instance-id", err, lager.Data{
			"event": "failed",
			"path":  redisConfigPath,
		})
		return "", err
	}

	instanceID := filepath.Base(dir)

	p.logger.Info("shared-instance-id", lager.Data{
		"event":       "done",
		"path":        redisConfigPath,
		"instance_id": instanceID,
	})

	return instanceID, nil
}
