package availability

import (
	"errors"
	"net"
	"time"
)

const checkInteral = time.Millisecond * 10

func Check(address *net.TCPAddr, timeout time.Duration) error {
	for elapsed := time.Duration(0); elapsed < timeout; elapsed = elapsed + checkInteral {
		if isListening(address) {
			return nil
		}
		time.Sleep(checkInteral)
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
