package s3bucket_test

import (
	"github.com/mitchellh/goamz/s3/s3test"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestS3bucket(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "S3bucket Suite")
}

var fakeS3EndpointURL string

var _ = BeforeSuite(func() {
	s3TestServerConfig := &s3test.Config{
		Send409Conflict: true,
	}
	s3testServer, err := s3test.NewServer(s3TestServerConfig)
	Ω(err).ToNot(HaveOccurred())
	fakeS3EndpointURL = s3testServer.URL()
})
