package backup

import (
	"errors"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/st3v/glager"

	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-cf/cf-redis-broker/recovery"
	"github.com/pivotal-cf/cf-redis-broker/recovery/task"
	redis "github.com/pivotal-cf/cf-redis-broker/redis/client"
	"github.com/pivotal-cf/cf-redis-broker/redis/client/fakes"
	"github.com/pivotal-golang/lager"
)

type fakeSnapshot struct {
	CreateErr     error
	CreateResult  task.Artifact
	CreateInvoked int
}

func (s *fakeSnapshot) Create() (task.Artifact, error) {
	s.CreateInvoked++
	return s.CreateResult, s.CreateErr
}

type fakeTask struct {
	TaskName               string
	RunInvokedWithArtifact []task.Artifact
	RunErr                 error
}

func (t *fakeTask) Name() string {
	return t.TaskName
}

func (t *fakeTask) Run(artifact task.Artifact) (task.Artifact, error) {
	if t.RunInvokedWithArtifact == nil {
		t.RunInvokedWithArtifact = []task.Artifact{}
	}

	t.RunInvokedWithArtifact = append(t.RunInvokedWithArtifact, artifact)
	return task.NewArtifact(t.Name()), t.RunErr
}

var _ = Describe("backup", func() {
	Describe(".Backup", func() {
		var (
			backupErr       error
			log             *gbytes.Buffer
			logger          lager.Logger
			snapshot        *fakeSnapshot
			initialArtifact task.Artifact
			renamer         *fakeTask
			s3Uploader      *fakeTask
			cleaner         *fakeTask
			redisClient     redis.Client
		)

		BeforeEach(func() {
			log = gbytes.NewBuffer()
			logger = lager.NewLogger("logger")
			logger.RegisterSink(lager.NewWriterSink(log, lager.INFO))

			initialArtifact = task.NewArtifact("path/to/artifact")

			snapshot = &fakeSnapshot{
				CreateResult: initialArtifact,
			}
			snapshotProvider = func(client redis.Client, timeout time.Duration, logger lager.Logger) recovery.Snapshot {
				return snapshot
			}

			renamer = &fakeTask{
				TaskName: "renamer",
			}
			renameProvider = func(string, lager.Logger) task.Task {
				return renamer
			}

			s3Uploader = &fakeTask{
				TaskName: "s3uploader",
			}
			s3UploadProvider = func(bucketName, targetPath, endpoint, key, secret string, logger lager.Logger, options ...task.S3UploadOption) task.Task {
				return s3Uploader
			}

			cleaner = &fakeTask{
				TaskName: "cleaner",
			}
			cleanupProvider = func(originalRdbPath, renamedRdbPath string, logger lager.Logger, options ...cleanupOption) task.Task {
				return cleaner
			}

			redisClient = &fakes.Client{
				Host: "test-host",
				Port: 1446,
			}
		})

		JustBeforeEach(func() {
			backupErr = Backup(
				redisClient,
				time.Second,
				"bucket-name",
				"target-path",
				"endpoint",
				"key",
				"secret",
				logger,
			)
		})

		It("does not return an error", func() {
			Expect(backupErr).ToNot(HaveOccurred())
		})

		It("takes a snapshot", func() {
			Expect(snapshot.CreateInvoked).To(Equal(1))
		})

		It("renames the original snapshot", func() {
			Expect(renamer.RunInvokedWithArtifact).To(HaveLen(1))
			Expect(renamer.RunInvokedWithArtifact[0]).To(Equal(initialArtifact))
		})

		It("uploads the renamed snapshot to s3", func() {
			expectedArtifact := task.NewArtifact(renamer.Name())
			Expect(s3Uploader.RunInvokedWithArtifact).To(HaveLen(1))
			Expect(s3Uploader.RunInvokedWithArtifact[0]).To(Equal(expectedArtifact))
		})

		It("cleans up", func() {
			expectedArtifact := task.NewArtifact(s3Uploader.Name())
			Expect(cleaner.RunInvokedWithArtifact).To(HaveLen(1))
			Expect(cleaner.RunInvokedWithArtifact[0]).To(Equal(expectedArtifact))
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
				snapshot = &fakeSnapshot{
					CreateErr: expectedErr,
				}
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
				Expect(renamer.RunInvokedWithArtifact).To(HaveLen(0))
			})

			It("does not upload anything to s3", func() {
				Expect(s3Uploader.RunInvokedWithArtifact).To(HaveLen(0))
			})

			It("does not cleanup", func() {
				Expect(cleaner.RunInvokedWithArtifact).To(HaveLen(0))
			})
		})

		Context("when renaming the snapshot fails", func() {
			var expectedErr = errors.New("rename-error")

			BeforeEach(func() {
				renamer = &fakeTask{
					RunErr: expectedErr,
				}
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
				Expect(s3Uploader.RunInvokedWithArtifact).To(HaveLen(0))
			})

			It("does cleanup", func() {
				Expect(cleaner.RunInvokedWithArtifact).To(HaveLen(1))
			})
		})

		Context("when uploading the snapshot fails", func() {
			var expectedErr = errors.New("upload-error")

			BeforeEach(func() {
				s3Uploader = &fakeTask{
					RunErr: expectedErr,
				}
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
				Expect(cleaner.RunInvokedWithArtifact).To(HaveLen(1))
			})
		})
	})
})
