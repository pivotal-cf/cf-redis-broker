package s3_test

import (
	"fmt"

	"github.com/mitchellh/goamz/aws"
	goamz "github.com/mitchellh/goamz/s3"
	"github.com/pivotal-cf/cf-redis-broker/s3"
	"code.cloudfoundry.org/lager"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Client", func() {
	var (
		fakeRegion        aws.Region
		goamzBucketClient *goamz.Bucket
		bucketName        string
		log               *gbytes.Buffer
		logger            lager.Logger
	)

	BeforeEach(func() {
		bucketName = "i_am_bucket"

		logger = lager.NewLogger("logger")
		log = gbytes.NewBuffer()
		logger.RegisterSink(lager.NewWriterSink(log, lager.INFO))

		fakeRegion = aws.Region{
			Name:                 "fake_region",
			S3Endpoint:           fakeS3EndpointURL,
			S3LocationConstraint: true,
		}
		goamzBucketClient = goamz.New(aws.Auth{}, fakeRegion).Bucket(bucketName)
	})

	Describe("GetOrCreateBucket", func() {
		Context("when the bucket already exists", func() {
			var (
				err    error
				bucket s3.Bucket
			)

			BeforeEach(func() {
				err := goamzBucketClient.PutBucket(goamz.BucketOwnerFull)
				Expect(err).NotTo(HaveOccurred())

				client := s3.NewClient(fakeS3EndpointURL, "accessKey", "secretKey", logger)
				bucket, err = client.GetOrCreateBucket(bucketName)
			})

			AfterEach(func() {
				err := goamzBucketClient.DelBucket()
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns the bucket", func() {
				Expect(bucket.Name()).To(Equal(bucketName))
			})

			It("does not return an error", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("provides logging", func() {
				Expect(log).To(gbytes.Say(
					fmt.Sprintf(`{"bucket_name":"%s","event":"starting"}`, bucketName),
				))
				Expect(log).To(gbytes.Say(
					fmt.Sprintf(`{"bucket_name":"%s","event":"already-exists"}`, bucketName),
				))
				Expect(log).To(gbytes.Say(
					fmt.Sprintf(`{"bucket_name":"%s","event":"done"}`, bucketName),
				))
			})
		})

		Context("when the bucket does not exist", func() {
			var (
				bucket    s3.Bucket
				createErr error
			)

			BeforeEach(func() {
				bucketList, err := goamzBucketClient.ListBuckets()
				Expect(err).NotTo(HaveOccurred())
				Expect(bucketList.Buckets).To(HaveLen(0))

				client := s3.NewClient(fakeS3EndpointURL, "accessKey", "secretKey", logger)
				bucket, createErr = client.GetOrCreateBucket(bucketName)
			})

			AfterEach(func() {
				err := goamzBucketClient.DelBucket()
				Expect(err).NotTo(HaveOccurred())
			})

			It("does not return an error", func() {
				Expect(createErr).NotTo(HaveOccurred())
			})

			It("returns the bucket", func() {
				Expect(bucket.Name()).To(Equal(bucketName))
			})

			It("provides logging", func() {
				Expect(log).To(gbytes.Say(
					fmt.Sprintf(`{"bucket_name":"%s","event":"starting"}`, bucketName),
				))
				Expect(log).To(gbytes.Say(
					fmt.Sprintf(`{"bucket_name":"%s","event":"done"}`, bucketName),
				))
			})

			It("creates the bucket", func() {
				bucketList, err := goamzBucketClient.ListBuckets()
				Expect(err).NotTo(HaveOccurred())
				Expect(bucketList.Buckets).To(HaveLen(1))
				Expect(bucketList.Buckets[0].Name).To(Equal(bucketName))
			})
		})

		Context("when the goamz client returns a goamz.Error", func() {
			var (
				createErr error
				errString = "The specified bucket is not valid"
			)

			BeforeEach(func() {
				bucketName = "@invalidBucketName#"
				client := s3.NewClient(fakeS3EndpointURL, "accessKey", "secretKey", logger)
				_, createErr = client.GetOrCreateBucket(bucketName)
			})

			It("returns the same error", func() {
				Expect(createErr.Error()).To(ContainSubstring(errString))
			})

			It("logs the error", func() {
				Expect(log).To(gbytes.Say(
					fmt.Sprintf(
						`{"bucket_name":"%s","error":"%s","event":"failed"}`,
						bucketName,
						errString,
					),
				))
			})
		})

		Context("when the goamz client returns a generic error", func() {
			var (
				createErr error
				errString = "unsupported protocol scheme"
			)

			BeforeEach(func() {
				client := s3.NewClient(
					"not-a-real-endpoint",
					"accessKey",
					"secretKey",
					logger,
				)
				_, createErr = client.GetOrCreateBucket(bucketName)
			})

			It("returns the same error", func() {
				Expect(createErr.Error()).To(ContainSubstring(errString))
			})

			It("logs the error", func() {
				Expect(log).To(gbytes.Say(
					fmt.Sprintf(
						`{"bucket_name":"%s","error":".*%s.*","event":"failed"}`,
						bucketName,
						errString,
					),
				))
			})
		})
	})
})
