package backup_test

import (
	"errors"
	"fmt"
	"time"

	"github.com/pivotal-cf/cf-redis-broker/recovery"
	"github.com/pivotal-cf/cf-redis-broker/recovery/task"
	"github.com/pivotal-cf/cf-redis-broker/redis/backup"
	"github.com/pivotal-cf/cf-redis-broker/redis/client/fakes"
	"github.com/pivotal-golang/lager"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Snapshotter", func() {
	Describe(".Snapshot", func() {
		var (
			artifact task.Artifact
			err      error

			snapshotter recovery.Snapshotter

			fakeRedisClient      *fakes.Client
			expectedArtifactPath = "the/artifact/path"
			logger               lager.Logger
			log                  *gbytes.Buffer
			timeout              time.Duration
		)

		BeforeEach(func() {
			fakeRedisClient = &fakes.Client{
				ExpectedRDBPath: expectedArtifactPath,
			}
			timeout = 123 * time.Second
			logger = lager.NewLogger("logger")
			log = gbytes.NewBuffer()
			logger.RegisterSink(lager.NewWriterSink(log, lager.INFO))
			snapshotter = backup.NewSnapshotter(fakeRedisClient, timeout, logger)
		})

		JustBeforeEach(func() {
			artifact, err = snapshotter.Snapshot()
		})

		It("creates an artifact with the right path", func() {
			Expect(artifact.Path()).To(Equal(expectedArtifactPath))
		})

		It("does not return an error", func() {
			Expect(err).ToNot(HaveOccurred())
		})

		It("runs bgsave on client and waits for completion of save", func() {
			Expect(fakeRedisClient.RunBGSaveCallCount).To(Equal(1))
			Expect(fakeRedisClient.WaitForNewSaveSinceCallCount).To(Equal(1))
		})

		It("provides logging", func() {
			Expect(log).To(gbytes.Say(fmt.Sprintf(`{"event":"starting","task":"create-snapshot","timeout":"%s"}`, timeout.String())))
			Expect(log).To(gbytes.Say(fmt.Sprintf(`{"event":"done","task":"create-snapshot","timeout":"%s"}`, timeout.String())))
			Expect(log).To(gbytes.Say(`{"event":"starting","task":"get-rdb-path"}`))
			Expect(log).To(gbytes.Say(fmt.Sprintf(`{"event":"done","path":"%s","task":"get-rdb-path"}`, expectedArtifactPath)))
		})

		Context("when run bgsave fails", func() {
			var expectedErr = errors.New("run-bgsave-error")

			BeforeEach(func() {
				fakeRedisClient.ExpectedRunGBSaveErr = expectedErr

				logger = lager.NewLogger("logger")
				log = gbytes.NewBuffer()
				logger.RegisterSink(lager.NewWriterSink(log, lager.INFO))

				snapshotter = backup.NewSnapshotter(fakeRedisClient, 123, logger)
			})

			It("keeps going and waits for completion of an existing save", func() {
				Expect(fakeRedisClient.WaitForNewSaveSinceCallCount).To(Equal(1))
			})

			It("logs the error", func() {
				Expect(log).To(gbytes.Say(fmt.Sprintf(`"error":"%s","event":"failed","task":"run-bg-save"}`, expectedErr.Error())))
			})

			Context("and the save operation never completes", func() {
				waitForNewSaveErr := errors.New("something went wrong!")

				BeforeEach(func() {
					fakeRedisClient.ExpectedWaitForNewSaveSinceErr = waitForNewSaveErr
				})

				It("errors", func() {
					Expect(err).To(MatchError(waitForNewSaveErr))
				})

				It("logs the error", func() {
					Expect(log).To(gbytes.Say(fmt.Sprintf(`error":"%s","event":"failed","last_time_save":0,"task":"wait-for-new-save","timeout":"123ns"}`, waitForNewSaveErr.Error())))
				})
			})
		})

		Context("when we do not get the snapshot path", func() {
			var expectedErr = errors.New("rdb-path-error")

			BeforeEach(func() {
				fakeRedisClient = &fakes.Client{
					ExpectedRDBPathErr: expectedErr,
				}

				logger = lager.NewLogger("logger")
				log = gbytes.NewBuffer()
				logger.RegisterSink(lager.NewWriterSink(log, lager.INFO))

				snapshotter = backup.NewSnapshotter(fakeRedisClient, 123, logger)
			})

			It("returns the error", func() {
				Expect(err).To(Equal(expectedErr))
			})

			It("logs the error", func() {
				Expect(log).To(gbytes.Say(fmt.Sprintf(`"error":"%s","event":"failed","task":"get-rdb-path"}`, err.Error())))
			})
		})
	})
})
