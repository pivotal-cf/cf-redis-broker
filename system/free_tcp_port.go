package system

import (
	"net"
	"strconv"
)

func FindFreePort() (int, error) {
	l, _ := net.Listen("tcp", ":0")
	defer l.Close()

	parsedPort, parseErr := strconv.ParseInt(l.Addr().String()[5:], 10, 32)
	if parseErr != nil {
		return -1, parseErr
	}

	return int(parsedPort), nil
}
