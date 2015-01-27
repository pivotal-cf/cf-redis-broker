package credentials

import (
	"errors"
	"strconv"

	"github.com/pivotal-cf/cf-redis-broker/redisconf"
)

type Credentials struct {
	Port     int    `json:"port"`
	Password string `json:"password"`
}

func Parse(configPath string) (Credentials, error) {
	conf, err := redisconf.Load(configPath)
	if err != nil {
		return Credentials{}, err
	}

	portStr := conf.Get("port")
	if portStr == "" {
		return Credentials{}, errors.New("No port found in redis.conf")
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return Credentials{}, err
	}

	password := conf.Get("requirepass")
	if password == "" {
		return Credentials{}, errors.New("No password found in redis.conf")
	}

	return Credentials{
		Port:     port,
		Password: password,
	}, nil
}
