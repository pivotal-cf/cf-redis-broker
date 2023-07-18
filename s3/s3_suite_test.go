package s3_test

import (
	"github.com/mitchellh/goamz/s3/s3test"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

var fakeS3EndpointURL string

func TestS3(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "S3 Suite")
}

var _ = BeforeSuite(func() {
	s3TestServerConfig := &s3test.Config{
		Send409Conflict: true,
	}
	s3testServer, err := s3test.NewServer(s3TestServerConfig)
	Ω(err).ToNot(HaveOccurred())
	fakeS3EndpointURL = s3testServer.URL()
})
