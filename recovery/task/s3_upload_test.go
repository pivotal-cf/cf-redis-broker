package task_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/st3v/glager"

	goamz "github.com/goamz/goamz/s3"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-cf/cf-redis-broker/recovery/task"
	"github.com/pivotal-cf/cf-redis-broker/s3"
	"code.cloudfoundry.org/lager"
)

type fakeS3Client struct {
	GetOrCreateBucketResult         s3.Bucket
	GetOrCreateBucketErr            error
	GetOrCreateBucketInvokedWithArg []string
}

func (c *fakeS3Client) GetOrCreateBucket(name string) (s3.Bucket, error) {
	if c.GetOrCreateBucketInvokedWithArg == nil {
		c.GetOrCreateBucketInvokedWithArg = []string{}
	}

	c.GetOrCreateBucketInvokedWithArg = append(c.GetOrCreateBucketInvokedWithArg, name)
	return c.GetOrCreateBucketResult, c.GetOrCreateBucketErr
}

func (c *fakeS3Client) ApiClient() *goamz.S3 { return nil }

type fakeS3Bucket struct {
	BucketName            string
	UploadErr             error
	UploadInvokedWithArgs []map[string]string
}

func (b *fakeS3Bucket) Name() string {
	return b.BucketName
}

func (b *fakeS3Bucket) Upload(source, target string) error {
	if b.UploadInvokedWithArgs == nil {
		b.UploadInvokedWithArgs = []map[string]string{}
	}

	b.UploadInvokedWithArgs = append(b.UploadInvokedWithArgs, map[string]string{
		"source": source,
		"target": target,
	})

	return b.UploadErr
}

var _ = Describe("S3Upload", func() {
	var (
		log    *gbytes.Buffer
		logger lager.Logger
	)

	BeforeEach(func() {
		log = gbytes.NewBuffer()
		logger = lager.NewLogger("redis")
		logger.RegisterSink(lager.NewWriterSink(log, lager.INFO))
	})

	Describe(".Name", func() {
		It("returns the correct name", func() {
			upload := task.NewS3Upload("bucket", "target", "endpoint", "key", "secret", logger)
			Expect(upload.Name()).To(Equal("s3upload"))
		})
	})

	Describe(".Run", func() {
		var (
			expectedSourcePath = "path/to/source"
			expectedTargetPath = "path/to/target"
			expectedBucketName = "some-bucket-name"
			runErr             error
			client             *fakeS3Client
			bucket             *fakeS3Bucket
			upload             task.Task
		)

		JustBeforeEach(func() {
			_, runErr = upload.Run(task.NewArtifact(expectedSourcePath))
		})

		BeforeEach(func() {
			bucket = &fakeS3Bucket{
				BucketName: expectedBucketName,
			}

			client = &fakeS3Client{
				GetOrCreateBucketResult: bucket,
			}

			upload = task.NewS3Upload(
				expectedBucketName,
				expectedTargetPath,
				"endpoint",
				"key",
				"secret",
				logger,
				task.InjectS3Client(client),
			)
		})

		It("creates or gets the S3 bucket", func() {
			Expect(client.GetOrCreateBucketInvokedWithArg).To(HaveLen(1))
			Expect(client.GetOrCreateBucketInvokedWithArg[0]).To(Equal(expectedBucketName))
		})

		It("uploads the artifact to the S3 bucket", func() {
			Expect(bucket.UploadInvokedWithArgs).To(HaveLen(1))
			Expect(bucket.UploadInvokedWithArgs[0]).To(Equal(map[string]string{
				"source": expectedSourcePath,
				"target": expectedTargetPath,
			}))
		})

		It("logs the upload", func() {
			Expect(log).To(ContainSequence(
				Info(
					Action("redis.s3upload"),
					Data("event", "starting", "bucket", bucket.BucketName, "source_path", expectedSourcePath, "target_path", expectedTargetPath),
				),
				Info(
					Action("redis.s3upload.create-bucket"),
					Data("event", "starting", "bucket", bucket.BucketName),
				),
				Info(
					Action("redis.s3upload.create-bucket"),
					Data("event", "done", "bucket", bucket.BucketName),
				),
				Info(
					Action("redis.s3upload.upload"),
					Data("event", "starting", "bucket", bucket.BucketName, "source_path", expectedSourcePath, "target_path", expectedTargetPath),
				),
				Info(
					Action("redis.s3upload.upload"),
					Data("event", "done", "bucket", bucket.BucketName, "source_path", expectedSourcePath, "target_path", expectedTargetPath),
				),
				Info(
					Action("redis.s3upload"),
					Data("event", "done", "bucket", bucket.BucketName, "source_path", expectedSourcePath, "target_path", expectedTargetPath),
				),
			))
		})

		Context("when creating the bucket fails", func() {
			var expectedErr = errors.New("bucket-creation-error")

			BeforeEach(func() {
				client.GetOrCreateBucketErr = expectedErr
			})

			It("returns the error", func() {
				Expect(runErr).To(Equal(expectedErr))
			})

			It("logs the error", func() {
				Expect(log).To(ContainSequence(
					Info(
						Action("redis.s3upload"),
						Data("event", "starting", "bucket", bucket.BucketName, "source_path", expectedSourcePath, "target_path", expectedTargetPath),
					),
					Info(
						Action("redis.s3upload.create-bucket"),
						Data("event", "starting", "bucket", bucket.BucketName),
					),
					Error(
						expectedErr,
						Action("redis.s3upload.create-bucket"),
						Data("event", "failed", "bucket", bucket.BucketName),
					),
					Error(
						expectedErr,
						Action("redis.s3upload"),
						Data("event", "failed", "bucket", bucket.BucketName, "source_path", expectedSourcePath, "target_path", expectedTargetPath),
					),
				))
			})
		})

		Context("when uploading to the bucket fails", func() {
			var expectedErr = errors.New("upload-error")

			BeforeEach(func() {
				bucket.UploadErr = expectedErr
			})

			It("returns the error", func() {
				Expect(runErr).To(Equal(expectedErr))
			})

			It("logs the error", func() {
				Expect(log).To(ContainSequence(
					Info(
						Action("redis.s3upload"),
						Data("event", "starting", "bucket", bucket.BucketName, "source_path", expectedSourcePath, "target_path", expectedTargetPath),
					),
					Info(
						Action("redis.s3upload.create-bucket"),
						Data("event", "starting", "bucket", bucket.BucketName),
					),
					Info(
						Action("redis.s3upload.create-bucket"),
						Data("event", "done", "bucket", bucket.BucketName),
					),
					Info(
						Action("redis.s3upload.upload"),
						Data("event", "starting", "bucket", bucket.BucketName, "source_path", expectedSourcePath, "target_path", expectedTargetPath),
					),
					Error(
						expectedErr,
						Action("redis.s3upload.upload"),
						Data("event", "failed", "bucket", bucket.BucketName, "source_path", expectedSourcePath, "target_path", expectedTargetPath),
					),
					Error(
						expectedErr,
						Action("redis.s3upload"),
						Data("event", "failed", "bucket", bucket.BucketName, "source_path", expectedSourcePath, "target_path", expectedTargetPath),
					),
				))
			})
		})
	})
})
