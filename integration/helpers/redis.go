package helpers

import (
	"fmt"

	redisclient "github.com/garyburd/redigo/redis"
	. "github.com/onsi/gomega"
)

func BuildRedisClient(port uint, host string, password string) redisclient.Conn {
	url := fmt.Sprintf("%s:%d", host, port)

	client, err := redisclient.Dial("tcp", url)
	Ω(err).NotTo(HaveOccurred())

	_, err = client.Do("AUTH", password)
	Ω(err).NotTo(HaveOccurred())

	return client
}
