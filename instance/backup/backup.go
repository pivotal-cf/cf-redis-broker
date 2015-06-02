package backup

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/pivotal-cf/cf-redis-broker/broker"
	"github.com/pivotal-cf/cf-redis-broker/redisconf"
)

type InstanceIDProvider interface {
	InstanceID(redisConfigPath, nodeIP string) (string, error)
}

func NewInstanceIDProvider(planName string) (InstanceIDProvider, error) {
	switch planName {
	case broker.PlanNameShared:
		return &sharedPlan{}, nil
	case broker.PlanNameDedicated:
		return &dedicatedPlan{}, nil
	}

	return nil, errors.New("Unknown plan name")
}

func LoadRedisConfigs(configRoot, configFilename string) ([]redisconf.Conf, error) {
	redisConfigPaths, err := findFiles(configRoot, configFilename)
	if err != nil {
		return nil, err
	}

	redisConfigs := make([]redisconf.Conf, len(redisConfigPaths))
	for i, path := range redisConfigPaths {
		redisConfigs[i], err = redisconf.Load(path)
		if err != nil {
			return nil, err
		}
	}

	return redisConfigs, err
}

func findFiles(rootPath, filename string) ([]string, error) {
	paths := []string{}

	matcher := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			if info.Name() == filename {
				paths = append(paths, path)
			}
		}
		return nil
	}

	if err := filepath.Walk(rootPath, matcher); err != nil {
		return nil, err
	}

	return paths, nil
}
