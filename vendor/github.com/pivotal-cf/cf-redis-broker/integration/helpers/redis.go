package helpers

import (
	"fmt"
	"strconv"

	"github.com/garyburd/redigo/redis"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/cf-redis-broker/redisconf"
)

func BuildRedisClient(port uint, host string, password string) redis.Conn {
	url := fmt.Sprintf("%s:%d", host, port)

	client, err := redis.Dial("tcp", url)
	Ω(err).NotTo(HaveOccurred())

	_, err = client.Do("AUTH", password)
	Ω(err).NotTo(HaveOccurred())

	return client
}

func BuildRedisClientFromConf(conf redisconf.Conf) redis.Conn {
	port, err := strconv.Atoi(conf.Get("port"))
	Ω(err).NotTo(HaveOccurred())

	password := conf.Get("requirepass")

	return BuildRedisClient(uint(port), "localhost", password)
}
