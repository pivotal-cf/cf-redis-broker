package system

import (
	"net"
	"regexp"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf/cf-redis-broker/system/fakes"
)

var _ = Describe("Next available TCP port", func() {
	Describe("FindFreePort", func() {
		It("finds a free TCP port", func() {
			port, _ := FindFreePort()
			portStr := strconv.Itoa(port)

			matched, err := regexp.MatchString("^[0-9]+$", portStr)
			Expect(err).NotTo(HaveOccurred())
			Expect(matched).To(Equal(true))

			l, err := net.Listen("tcp", ":"+portStr)
			Expect(err).NotTo(HaveOccurred())
			l.Close()
		})
	})

	Describe("getPortFromAddr", func() {
		var (
			port    int
			portErr error
			address string
			netAddr *fakes.AddrFake
		)

		BeforeEach(func() {
			netAddr = new(fakes.AddrFake)
			address = "192.168.0.1:8080"
		})

		JustBeforeEach(func() {
			netAddr.StringReturns(address)
			port, portErr = getPortFromAddr(netAddr)
		})

		It("does not return an error", func() {
			Expect(portErr).NotTo(HaveOccurred())
		})

		It("parses the address", func() {
			Expect(port).To(Equal(8080))
		})

		Context("when parsing an ipv6 address", func() {
			BeforeEach(func() {
				address = "[2001:db8::1]:9090"
			})

			It("does not return an error", func() {
				Expect(portErr).NotTo(HaveOccurred())
			})

			It("parses the address", func() {
				Expect(port).To(Equal(9090))
			})
		})

		Context("when address is malformed", func() {
			BeforeEach(func() {
				address = "foo"
			})

			It("returns an error", func() {
				Expect(portErr).To(HaveOccurred())
			})
		})

		Context("when address is empty", func() {
			BeforeEach(func() {
				address = ""
			})

			It("returns an error", func() {
				Expect(portErr).To(HaveOccurred())
			})
		})
	})
})
