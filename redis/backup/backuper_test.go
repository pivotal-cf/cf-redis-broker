package backup_test

import (
	"errors"
	"io/ioutil"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/st3v/glager"

	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-cf/cf-redis-broker/recovery/task"
	"github.com/pivotal-cf/cf-redis-broker/redis/backup"
	"github.com/pivotal-cf/cf-redis-broker/redis/backup/fakes"
	"github.com/pivotal-golang/lager"
)

var _ = Describe("RedisBackuper", func() {
	Describe(".Backup", func() {
		var (
			backupErr       error
			log             *gbytes.Buffer
			tmpDir          string
			logger          lager.Logger
			providerFactory *fakes.FakeProviderFactory
			snapshotter     *fakes.FakeSnapshotter
			initialArtifact task.Artifact
			renameTask      *fakes.FakeTask
			s3UploadTask    *fakes.FakeTask
			cleanupTask     *fakes.FakeTask
			redisClient     *fakes.FakeRedisClient
			backuper        backup.RedisBackuper

			expectedTimeout    = 123 * time.Second
			expectedBucketName = "some-bucket-name"
			expectedTargetPath = "some-target-path"
			expectedEndpoint   = "some-endpoint"
			expectedAccessKey  = "some-access-key"
			expectedSecretKey  = "some-secret-key"
		)

		BeforeEach(func() {
			log = gbytes.NewBuffer()
			logger = lager.NewLogger("logger")
			logger.RegisterSink(lager.NewWriterSink(log, lager.INFO))

			initialArtifact = task.NewArtifact("path/to/artifact")

			providerFactory = new(fakes.FakeProviderFactory)

			snapshotter = new(fakes.FakeSnapshotter)
			snapshotter.SnapshotReturns(initialArtifact, nil)
			providerFactory.SnapshotterProviderReturns(snapshotter)

			renameTask = new(fakes.FakeTask)
			renameTask.NameReturns("rename")
			renameTask.RunReturns(task.NewArtifact("rename"), nil)
			providerFactory.RenameTaskProviderReturns(renameTask)

			s3UploadTask = new(fakes.FakeTask)
			s3UploadTask.NameReturns("s3upload")
			s3UploadTask.RunReturns(task.NewArtifact("s3upload"), nil)
			providerFactory.S3UploadTaskProviderReturns(s3UploadTask)

			cleanupTask = new(fakes.FakeTask)
			cleanupTask.NameReturns("cleaner")
			cleanupTask.RunReturns(task.NewArtifact("cleanup"), nil)
			providerFactory.CleanupTaskProviderReturns(cleanupTask)

			redisClient = new(fakes.FakeRedisClient)
			redisClient.AddressReturns("test-host:1446")

			var err error
			tmpDir, err = ioutil.TempDir("", "redis-backups-test")
			Expect(err).NotTo(HaveOccurred())

			backuper = backup.NewRedisBackuper(
				expectedTimeout,
				expectedBucketName,
				expectedEndpoint,
				expectedAccessKey,
				expectedSecretKey,
				tmpDir,
				logger,
				backup.InjectSnapshotterProvider(providerFactory.SnapshotterProvider),
				backup.InjectRenameTaskProvider(providerFactory.RenameTaskProvider),
				backup.InjectS3UploadTaskProvider(providerFactory.S3UploadTaskProvider),
				backup.InjectCleanupTaskProvider(providerFactory.CleanupTaskProvider),
			)
		})

		AfterEach(func() {
			Expect(os.RemoveAll(tmpDir)).To(Succeed())
		})

		JustBeforeEach(func() {
			backupErr = backuper.Backup(
				redisClient,
				expectedTargetPath,
			)
		})

		It("does not return an error", func() {
			Expect(backupErr).ToNot(HaveOccurred())
		})

		It("creates the snapshotter with the correct client", func() {
			actualClient, _, _ := providerFactory.SnapshotterProviderArgsForCall(0)
			Expect(actualClient).To(Equal(redisClient))
		})

		It("creates the snapshotter with the correct timeout", func() {
			_, actualTimeout, _ := providerFactory.SnapshotterProviderArgsForCall(0)
			Expect(actualTimeout).To(Equal(expectedTimeout))
		})

		It("creates the snapshotter with the correct logger", func() {
			_, _, actualLogger := providerFactory.SnapshotterProviderArgsForCall(0)
			Expect(actualLogger).To(Equal(logger))
		})

		It("takes a snapshot", func() {
			Expect(snapshotter.SnapshotCallCount()).To(Equal(1))
		})

		It("creates the rename task with a new path that is different from the initial artifact", func() {
			newPath, _ := providerFactory.RenameTaskProviderArgsForCall(0)
			Expect(newPath).ToNot(Equal(initialArtifact.Path()))
		})

		It("creates the rename task with the correct logger", func() {
			_, actualLogger := providerFactory.RenameTaskProviderArgsForCall(0)
			Expect(actualLogger).To(Equal(logger))
		})

		It("creates the s3 upload task with the correct bucket name", func() {
			actualBucketName, _, _, _, _, _, _ := providerFactory.S3UploadTaskProviderArgsForCall(0)
			Expect(actualBucketName).To(Equal(expectedBucketName))
		})

		It("creates the s3 upload task with the correct target path", func() {
			_, actualTargetPath, _, _, _, _, _ := providerFactory.S3UploadTaskProviderArgsForCall(0)
			Expect(actualTargetPath).To(Equal(expectedTargetPath))
		})

		It("creates the s3 upload task with the correct endpoint", func() {
			_, _, actualEndpoint, _, _, _, _ := providerFactory.S3UploadTaskProviderArgsForCall(0)
			Expect(actualEndpoint).To(Equal(expectedEndpoint))
		})

		It("creates the s3 upload task with the correct access key", func() {
			_, _, _, actualAccessKey, _, _, _ := providerFactory.S3UploadTaskProviderArgsForCall(0)
			Expect(actualAccessKey).To(Equal(expectedAccessKey))
		})

		It("creates the s3 upload task with the correct endpoint", func() {
			_, _, _, _, actualSecretKey, _, _ := providerFactory.S3UploadTaskProviderArgsForCall(0)
			Expect(actualSecretKey).To(Equal(expectedSecretKey))
		})

		It("creates the s3 upload task with the correct logger", func() {
			_, _, _, _, _, actualLogger, _ := providerFactory.S3UploadTaskProviderArgsForCall(0)
			Expect(actualLogger).To(Equal(logger))
		})

		It("does not inject anything into the s3 upload task", func() {
			_, _, _, _, _, _, injectors := providerFactory.S3UploadTaskProviderArgsForCall(0)
			Expect(injectors).To(BeEmpty())
		})

		It("creates the cleanup task with the correct original path", func() {
			originalPath, _, _, _ := providerFactory.CleanupTaskProviderArgsForCall(0)
			Expect(originalPath).To(Equal(initialArtifact.Path()))
		})

		It("creates the cleanup task with the correct temporary path", func() {
			expectedTempPath, _ := providerFactory.RenameTaskProviderArgsForCall(0)
			_, tempPath, _, _ := providerFactory.CleanupTaskProviderArgsForCall(0)
			Expect(tempPath).To(Equal(expectedTempPath))
		})

		It("creates the cleanup task with the correct logger", func() {
			_, _, actualLogger, _ := providerFactory.CleanupTaskProviderArgsForCall(0)
			Expect(actualLogger).To(Equal(logger))
		})

		It("does not inject anything into the cleanup task", func() {
			_, _, _, injectors := providerFactory.CleanupTaskProviderArgsForCall(0)
			Expect(injectors).To(BeEmpty())
		})

		It("renames the original snapshot", func() {
			Expect(renameTask.RunCallCount()).To(Equal(1))
			Expect(renameTask.RunArgsForCall(0)).To(Equal(initialArtifact))
		})

		It("uploads the renamed snapshot to s3", func() {
			expectedArtifact := task.NewArtifact(renameTask.Name())
			Expect(s3UploadTask.RunCallCount()).To(Equal(1))
			Expect(s3UploadTask.RunArgsForCall(0)).To(Equal(expectedArtifact))
		})

		It("cleans up", func() {
			expectedArtifact := task.NewArtifact(s3UploadTask.Name())
			Expect(cleanupTask.RunCallCount()).To(Equal(1))
			Expect(cleanupTask.RunArgsForCall(0)).To(Equal(expectedArtifact))
		})

		It("logs", func() {
			Expect(log).To(ContainSequence(
				Info(
					Action("logger.backup"),
					Data("event", "starting"),
					Data("redis_address", redisClient.Address()),
				),
				Info(
					Action("logger.backup"),
					Data("event", "done"),
					Data("redis_address", redisClient.Address()),
				),
			))
		})

		Context("when snapshotting fails", func() {
			var expectedErr = errors.New("snapshot-error")

			BeforeEach(func() {
				snapshotter.SnapshotReturns(nil, expectedErr)
			})

			It("returns the error", func() {
				Expect(backupErr).To(Equal(expectedErr))
			})

			It("logs the error", func() {
				Expect(log).To(ContainSequence(
					Info(
						Action("logger.backup"),
						Data("event", "starting"),
						Data("redis_address", redisClient.Address()),
					),
					Error(
						expectedErr,
						Action("logger.backup"),
						Data("event", "failed"),
						Data("redis_address", redisClient.Address()),
					),
				))
			})

			It("does not rename anything", func() {
				Expect(renameTask.RunCallCount()).To(Equal(0))
			})

			It("does not upload anything to s3", func() {
				Expect(s3UploadTask.RunCallCount()).To(Equal(0))
			})

			It("does not cleanup", func() {
				Expect(cleanupTask.RunCallCount()).To(Equal(0))
			})
		})

		Context("when renaming the snapshot fails", func() {
			var expectedErr = errors.New("rename-error")

			BeforeEach(func() {
				renameTask.RunReturns(nil, expectedErr)
			})

			It("returns the error", func() {
				Expect(backupErr).To(Equal(expectedErr))
			})

			It("logs the error", func() {
				Expect(log).To(ContainSequence(
					Info(
						Action("logger.backup"),
						Data("event", "starting"),
						Data("redis_address", redisClient.Address()),
					),
					Error(
						expectedErr,
						Action("logger.backup"),
						Data("event", "failed"),
						Data("redis_address", redisClient.Address()),
					),
				))
			})

			It("does not upload anything to s3", func() {
				Expect(s3UploadTask.RunCallCount()).To(Equal(0))
			})

			It("does cleanup", func() {
				Expect(cleanupTask.RunCallCount()).To(Equal(1))
			})
		})

		Context("when uploading the snapshot fails", func() {
			var expectedErr = errors.New("upload-error")

			BeforeEach(func() {
				s3UploadTask.RunReturns(nil, expectedErr)
			})

			It("returns the error", func() {
				Expect(backupErr).To(Equal(expectedErr))
			})

			It("logs the error", func() {
				Expect(log).To(ContainSequence(
					Info(
						Action("logger.backup"),
						Data("event", "starting"),
						Data("redis_address", redisClient.Address()),
					),
					Error(
						expectedErr,
						Action("logger.backup"),
						Data("event", "failed"),
						Data("redis_address", redisClient.Address()),
					),
				))
			})

			It("does cleanup", func() {
				Expect(cleanupTask.RunCallCount()).To(Equal(1))
			})
		})
	})
})
