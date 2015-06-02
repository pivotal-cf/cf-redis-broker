package instance

import (
	"os"
	"path/filepath"

	"github.com/pivotal-cf/cf-redis-broker/redisconf"
)

func RedisConfigs(configRoot, configFilename string) (map[string]redisconf.Conf, error) {
	redisConfigPaths, err := findFiles(configRoot, configFilename)
	if err != nil {
		return nil, err
	}

	redisConfigs := map[string]redisconf.Conf{}
	for _, path := range redisConfigPaths {
		redisConfigs[path], err = redisconf.Load(path)
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
