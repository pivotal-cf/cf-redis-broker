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

	if err = availability.Check(address, 10*time.Second); err != nil {
		return false
	}

	return true
}
