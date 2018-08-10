package system

import (
	"net"
	"strconv"
	"strings"
)

func FindFreePort() (int, error) {
	l, _ := net.Listen("tcp", ":0")
	defer l.Close()

	parsedPort, parseErr := getPortFromAddr(l.Addr())
	if parseErr != nil {
		return -1, parseErr
	}

	return int(parsedPort), nil
}

func getPortFromAddr(addr net.Addr) (int, error) {
	tokens := strings.Split(addr.String(), ":")
	port := tokens[len(tokens)-1]

	parsedPort, parseErr := strconv.ParseInt(port, 10, 32)
	return int(parsedPort), parseErr
}
