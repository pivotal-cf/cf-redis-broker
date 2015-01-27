package availability

import (
	"net"
	"time"
)

func Check(address *net.TCPAddr, timeout time.Duration) error {
	action := func(successChan chan<- struct{}, terminate <-chan struct{}) {
		for {
			select {
			case <-terminate:
				return
			case <-time.After(10 * time.Millisecond):
				if isListening(address) {
					close(successChan)
					return
				}
			}
		}
	}
	enforcer := DeadlineEnforcer{
		Action: action,
	}
	return enforcer.DoWithin(timeout)
}

func isListening(address *net.TCPAddr) bool {
	_, err := net.DialTCP("tcp", nil, address)
	return err == nil
}
