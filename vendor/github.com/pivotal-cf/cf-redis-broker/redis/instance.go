package redis

import "net"

type Instance struct {
	ID       string
	Host     string
	Port     int
	Password string
}

func (instance Instance) Address() *net.TCPAddr {
	return &net.TCPAddr{
		IP:   net.ParseIP(instance.Host),
		Port: instance.Port,
	}
}
