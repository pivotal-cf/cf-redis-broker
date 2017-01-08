package availability_test

import (
	"net"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf/cf-redis-broker/availability"
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
