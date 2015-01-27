package config

import (
	"fmt"

	"github.com/pivotal-cf/cf-redis-broker/redisconf"
)

func SaveRedisConfAdditions(fromPath string, toPath string, syslogIdentSuffix string) error {
	defaultConfig, err := redisconf.Load(fromPath)
	if err != nil {
		return err
	}

	defaultConfig.Set("syslog-enabled", "yes")
	defaultConfig.Set("syslog-ident", fmt.Sprintf("redis-server-%s", syslogIdentSuffix))
	defaultConfig.Set("syslog-facility", "local0")

	err = defaultConfig.Save(toPath)
	if err != nil {
		return err
	}

	return nil
}
