package availability_test

import (
	"github.com/pivotal-cf/cf-redis-broker/availability"
	"net"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func listen(address net.TCPAddr, delayBeforeListening time.Duration, terminate, closed chan struct{}) {
	listener := listenTCP("tcp", &address)
	go func() {
		select {
		case <-terminate:
			listener.Close()
			close(closed)
		}
	}()
	time.Sleep(delayBeforeListening)
	listener.AcceptTCP()
}

func listenTLS(address net.TCPAddr, delayBeforeListening time.Duration, terminate, closed chan struct{}) {
	listener := listenTCPTLS("tcp", &address)
	go func() {
		select {
		case <-terminate:
			listener.Close()
			close(closed)
		}
	}()
	time.Sleep(delayBeforeListening)
	accept, err := listener.Accept()
	Expect(err).NotTo(HaveOccurred())
	accept.Write([]byte{1})
	accept.Close()
}

var _ = Describe("waiting for a port to become available", func() {
	var address *net.TCPAddr

	BeforeEach(func() {
		address = resolveTCPAddr("tcp", "localhost:19000")
	})

	Context("when nothing is listening at the specified address", func() {
		It("errors", func() {
			err := availability.Check(address, time.Second*1)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("when something is already listening at the specified address", func() {
		terminate := make(chan struct{})
		closed := make(chan struct{})

		BeforeEach(func() {
			go func() {
				defer GinkgoRecover()
				listen(*address, 0, terminate, closed)
			}()
		})

		AfterEach(func() {
			close(terminate)
			Eventually(closed).Should(BeClosed())
		})

		It("does not error", func() {
			err := availability.Check(address, time.Second*1)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when something begins to listen on address before timeout is up", func() {
		terminate := make(chan struct{})
		closed := make(chan struct{})

		BeforeEach(func() {
			go func() {
				defer GinkgoRecover()

				listen(*address, 500*time.Millisecond, terminate, closed)
			}()
		})

		AfterEach(func() {
			close(terminate)
			Eventually(closed).Should(BeClosed())
		})

		It("does not error", func() {
			err := availability.Check(address, time.Second*1)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

var _ = Describe("waiting for a tls port to become available", func() {
	var address *net.TCPAddr

	BeforeEach(func() {
		address = resolveTCPAddr("tcp", "localhost:19001")
	})

	Context("when nothing is listening at the specified address", func() {
		It("errors", func() {
			err := availability.CheckTLS(address, time.Second*1)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("when something is already listening at the specified address", func() {
		terminate := make(chan struct{})
		closed := make(chan struct{})

		BeforeEach(func() {
			go func() {
				defer GinkgoRecover()
				listenTLS(*address, 0, terminate, closed)
			}()
		})

		AfterEach(func() {
			close(terminate)
			Eventually(closed).Should(BeClosed())
		})

		It("does not error", func() {
			err := availability.CheckTLS(address, time.Second*1)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when something begins to listen on address before timeout is up", func() {
		terminate := make(chan struct{})
		closed := make(chan struct{})

		BeforeEach(func() {
			go func() {
				defer GinkgoRecover()

				listenTLS(*address, 500*time.Millisecond, terminate, closed)
			}()
		})

		AfterEach(func() {
			close(terminate)
			Eventually(closed).Should(BeClosed())
		})

		It("does not error", func() {
			err := availability.CheckTLS(address, time.Second*1)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
