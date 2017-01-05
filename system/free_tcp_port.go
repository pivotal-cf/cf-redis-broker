package system

import (
	"net"
	"strconv"
	"errors"
)

const (
	MIN_ACCEPTED_PORT int = 1024
	MAX_ACCEPTED_PORT int = 65535
)

type FreeTcpPort interface {
	FindFreePortInRange(minport int, maxport int) (int, error)
}
type FreeRangeTcpPort struct {
	FreeTcpPort
	IsPortAvailable func(num int) bool
}

func NewFreeTcpPort() FreeTcpPort {
	return &FreeRangeTcpPort{IsPortAvailable: isPortAvailable}
}
func isPortAvailable(num int) bool {
	l, err := net.Listen("tcp", ":" + strconv.Itoa(num))
	if err != nil {
		return false
	}
	l.Close()
	return true
}
func (f FreeRangeTcpPort) FindFreePortInRange(minport int, maxport int) (int, error) {
	port := minport
	for port <= maxport {
		if f.IsPortAvailable(port) {
			return port, nil
		}
		port++
	}
	return -1, errors.New("No port is available in the range. Please ask help to your Operator")
}
