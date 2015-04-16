package s3bucket_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf-experimental/goamz/s3/s3test"

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
