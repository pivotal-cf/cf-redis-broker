package availability_test

import (
	"crypto/tls"
	"net"
	"path"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
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

func listenTCPTLS(network string, laddr *net.TCPAddr) net.Listener {
	certFile, err := filepath.Abs(path.Join("assets", "tls", "server.crt"))
	Expect(err).ToNot(HaveOccurred())
	keyFile, err := filepath.Abs(path.Join("assets", "tls", "server.key"))
	Expect(err).ToNot(HaveOccurred())
	cer, err := tls.LoadX509KeyPair(certFile, keyFile)
	Expect(err).ToNot(HaveOccurred())
	config := &tls.Config{Certificates: []tls.Certificate{cer}}
	ln, err := tls.Listen(network, laddr.String(), config)
	Expect(err).ToNot(HaveOccurred())
	return ln
}
