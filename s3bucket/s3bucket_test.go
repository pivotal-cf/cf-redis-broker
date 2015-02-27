package s3bucket_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
	"github.com/pivotal-cf/cf-redis-broker/s3bucket"
)

var _ = Describe("s3bucket", func() {
	var fakeRegion aws.Region
	var goamzBucketClient *s3.Bucket
	var bucketName = "i_am_bucket"

	BeforeEach(func() {
		fakeRegion = aws.Region{
			Name:                 "rake_region",
			S3Endpoint:           fakeS3EndpointURL,
			S3LocationConstraint: true,
		}
		goamzBucketClient = s3.New(aws.Auth{}, fakeRegion).Bucket(bucketName)
	})

	Describe("GetOrCreate", func() {
		Context("when the bucket already exists", func() {
			BeforeEach(func() {
				err := goamzBucketClient.PutBucket(s3.BucketOwnerFull)
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				err := goamzBucketClient.DelBucket()
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns the bucket and no error", func() {
				client := s3bucket.NewClient(fakeS3EndpointURL, "region", "accessKey", "secretKey")
				bucket, err := client.GetOrCreate(bucketName)
				Expect(err).NotTo(HaveOccurred())
				Expect(bucket.Name).To(Equal(bucketName))
			})
		})

		Context("when the bucket does not exist", func() {
			It("creates the bucket", func() {
				bucketList, err := goamzBucketClient.ListBuckets()
				Expect(err).NotTo(HaveOccurred())
				Expect(bucketList.Buckets).To(HaveLen(0))

				client := s3bucket.NewClient(fakeS3EndpointURL, "region", "accessKey", "secretKey")
				_, err = client.GetOrCreate(bucketName)
				Expect(err).NotTo(HaveOccurred())

				bucketList, err = goamzBucketClient.ListBuckets()
				Expect(err).NotTo(HaveOccurred())
				Expect(bucketList.Buckets).To(HaveLen(1))
				Expect(bucketList.Buckets[0].Name).To(Equal(bucketName))
			})

			It("returns the bucket and no error", func() {
				client := s3bucket.NewClient(fakeS3EndpointURL, "region", "accessKey", "secretKey")
				bucket, err := client.GetOrCreate(bucketName)
				Expect(err).NotTo(HaveOccurred())
				Expect(bucket.Name).To(Equal(bucketName))
			})
		})

		Context("when goamz returns an error", func() {
			It("returns the same error", func() {
				client := s3bucket.NewClient("not-a-real-endpoint", "region", "accessKey", "secretKey")
				_, err := client.GetOrCreate(bucketName)
				Expect(err).To(MatchError(ContainSubstring("unsupported protocol scheme")))
			})
		})
	})

	Describe("Upload", func() {
		var bucket s3bucket.Bucket
		var path = "some/test/path"
		var data = []byte("some test data")

		BeforeEach(func() {
			var err error
			bucket, err = s3bucket.NewClient(fakeS3EndpointURL, "region", "accessKey", "secretKey").GetOrCreate(bucketName)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when everything works", func() {
			AfterEach(func() {
				err := goamzBucketClient.Del(path)
				Expect(err).NotTo(HaveOccurred())
				err = goamzBucketClient.DelBucket()
				Expect(err).NotTo(HaveOccurred())
			})

			It("uploads data to the correct path", func() {
				err := bucket.Upload(data, path)
				Expect(err).NotTo(HaveOccurred())

				content, err := goamzBucketClient.Get(path)
				Expect(content).To(Equal(data))
			})
		})

		Context("when goamz returns an error", func() {
			BeforeEach(func() {
				err := goamzBucketClient.DelBucket()
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns the same error", func() {
				err := bucket.Upload(data, path)
				Expect(err).To(MatchError("The specified bucket does not exist"))
			})
		})
	})
})
