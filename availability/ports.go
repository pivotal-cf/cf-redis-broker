package availability

import (
	"errors"
	"net"
	"time"
)

func Check(address *net.TCPAddr, timeout time.Duration) error {
	interval := time.Millisecond * 10
	for elapsed := time.Duration(0); elapsed < timeout; elapsed = elapsed + interval {
		if isListening(address) {
			return nil
		}

		time.Sleep(interval)
	}

	return errors.New("timeout")
}

func isListening(address *net.TCPAddr) bool {
	connection, err := net.DialTCP("tcp", nil, address)
	if connection != nil {
		connection.Close()
	}
	return err == nil
}
