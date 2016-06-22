package instance

import (
	"os"
	"path/filepath"

	"github.com/pivotal-cf/cf-redis-broker/redisconf"
)

type RedisConfig struct {
	Path string
	Conf redisconf.Conf
}

type RedisConfigFinder interface {
	Find() ([]RedisConfig, error)
}

func NewRedisConfigFinder(rootPath, configFilename string) RedisConfigFinder {
	return &configFinder{
		root:     rootPath,
		filename: configFilename,
	}
}

type configFinder struct {
	root     string
	filename string
}

func (f *configFinder) Find() ([]RedisConfig, error) {
	paths, err := f.findFiles()
	if err != nil {
		return nil, err
	}

	redisConfigs := make([]RedisConfig, len(paths))

	for i, path := range paths {
		config, err := redisconf.Load(path)
		if err != nil {
			return nil, err
		}

		redisConfigs[i] = RedisConfig{
			Conf: config,
			Path: path,
		}
	}

	return redisConfigs, err
}

func (f *configFinder) findFiles() ([]string, error) {
	paths := []string{}

	matcher := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			if info.Name() == f.filename {
				paths = append(paths, path)
			}
		}
		return nil
	}

	if err := filepath.Walk(f.root, matcher); err != nil {
		return nil, err
	}

	return paths, nil
}
