package backup_test

import (
	"errors"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/st3v/glager"

	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-cf/cf-redis-broker/recovery"
	"github.com/pivotal-cf/cf-redis-broker/recovery/task"
	"github.com/pivotal-cf/cf-redis-broker/redis/backup"
	"github.com/pivotal-cf/cf-redis-broker/redis/backup/fakes"
	redis "github.com/pivotal-cf/cf-redis-broker/redis/client"
	"github.com/pivotal-golang/lager"
)

var _ = Describe("RedisBackuper", func() {
	Describe(".Backup", func() {
		var (
			backupErr       error
			log             *gbytes.Buffer
			logger          lager.Logger
			snapshotter     *fakes.FakeSnapshotter
			initialArtifact task.Artifact
			renameTask      *fakes.FakeTask
			s3UploadTask    *fakes.FakeTask
			cleanupTask     *fakes.FakeTask
			redisClient     *fakes.FakeRedisClient
			backuper        backup.RedisBackuper
		)

		BeforeEach(func() {
			log = gbytes.NewBuffer()
			logger = lager.NewLogger("logger")
			logger.RegisterSink(lager.NewWriterSink(log, lager.INFO))

			initialArtifact = task.NewArtifact("path/to/artifact")

			snapshotter = new(fakes.FakeSnapshotter)
			snapshotter.SnapshotReturns(initialArtifact, nil)
			snapshotterProvider := func(redis.Client, time.Duration, lager.Logger) recovery.Snapshotter {
				return snapshotter
			}

			renameTask = new(fakes.FakeTask)
			renameTask.NameReturns("rename")
			renameTask.RunReturns(task.NewArtifact("rename"), nil)
			renameTaskProvider := func(string, lager.Logger) task.Task {
				return renameTask
			}

			s3UploadTask = new(fakes.FakeTask)
			s3UploadTask.NameReturns("s3upload")
			s3UploadTask.RunReturns(task.NewArtifact("s3upload"), nil)
			s3UploadTaskProvider := func(
				string, string, string, string, string, lager.Logger, ...task.S3UploadInjector,
			) task.Task {
				return s3UploadTask
			}

			cleanupTask = new(fakes.FakeTask)
			cleanupTask.NameReturns("cleaner")
			cleanupTask.RunReturns(task.NewArtifact("cleanup"), nil)
			cleanupTaskProvider := func(
				string, string, lager.Logger, ...backup.CleanupInjector,
			) task.Task {
				return cleanupTask
			}

			redisClient = new(fakes.FakeRedisClient)
			redisClient.AddressReturns("test-host:1446")

			backuper = backup.NewRedisBackuper(
				time.Second,
				"bucket-name",
				"endpoint",
				"key",
				"secret",
				logger,
				backup.InjectSnapshotterProvider(snapshotterProvider),
				backup.InjectRenameTaskProvider(renameTaskProvider),
				backup.InjectS3UploadTaskProvider(s3UploadTaskProvider),
				backup.InjectCleanupTaskProvider(cleanupTaskProvider),
			)
		})

		JustBeforeEach(func() {
			backupErr = backuper.Backup(
				redisClient,
				"target-path",
			)
		})

		It("does not return an error", func() {
			Expect(backupErr).ToNot(HaveOccurred())
		})

		It("takes a snapshot", func() {
			Expect(snapshotter.SnapshotCallCount()).To(Equal(1))
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
