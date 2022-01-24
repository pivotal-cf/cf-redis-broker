package helpers

import (
	"fmt"
	"net"
	"time"

	"github.com/pivotal-cf/cf-redis-broker/availability"
)

func ServiceAvailable(port uint) bool {
	address, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		return false
	}

	return availability.Check(address, 10*time.Second) == nil
}

func ServiceAvailableTLS(tlsPort uint) bool {
	address, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("localhost:%d", tlsPort))
	if err != nil {
		return false
	}

	return availability.CheckTLS(address, 10*time.Second) == nil
}
