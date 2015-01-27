package availability_test

import (
	"net"
	"time"

	"github.com/pivotal-cf/cf-redis-broker/availability"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func listen(address net.TCPAddr, delayBeforeListening time.Duration, terminate, closed chan struct{}) {
	listener, err := net.ListenTCP("tcp", &address)
	Ω(err).ShouldNot(HaveOccurred())
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

var _ = Describe("waiting for a port to become available", func() {

	var address *net.TCPAddr

	BeforeEach(func() {
		var err error
		address, err = net.ResolveTCPAddr("tcp", "localhost:19000")
		Ω(err).ShouldNot(HaveOccurred())
	})

	Context("when nothing is listening at the specified address", func() {
		It("errors", func() {
			err := availability.Check(address, time.Second*1)
			Ω(err).Should(HaveOccurred())
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
			Ω(err).ShouldNot(HaveOccurred())
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
			Ω(err).ShouldNot(HaveOccurred())
		})
	})
})
