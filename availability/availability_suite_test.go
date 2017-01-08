package availability_test

import (
	"net"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestAvailability(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Availability Suite")
}

func resolveTCPAddr(network, addr string) *net.TCPAddr {
	address, err := net.ResolveTCPAddr(network, addr)
	Expect(err).NotTo(HaveOccurred())
	return address
}

func listenTCP(network string, laddr *net.TCPAddr) *net.TCPListener {
	listener, err := net.ListenTCP(network, laddr)
	Expect(err).NotTo(HaveOccurred())
	return listener
}
