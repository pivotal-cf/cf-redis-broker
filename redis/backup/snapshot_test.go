package backup_test

import (
	"errors"
	"fmt"
	"time"

	"github.com/pivotal-cf/cf-redis-broker/recovery/task"
	"github.com/pivotal-cf/cf-redis-broker/redis/backup"
	"github.com/pivotal-cf/cf-redis-broker/redis/client/fakes"
	"github.com/pivotal-golang/lager"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Snapshot", func() {
	Describe(".Create", func() {
		var (
			artifact             task.Artifact
			err                  error
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
			snapshot := backup.NewSnapshot(fakeRedisClient, timeout, logger)
			artifact, err = snapshot.Create()
		})

		It("creates an artifact with the right path", func() {
			Expect(artifact.Path()).To(Equal(expectedArtifactPath))
		})

		It("does not return an error", func() {
			Expect(err).ToNot(HaveOccurred())
		})

		It("triggers create snapshot on the client", func() {
			Expect(fakeRedisClient.InvokedCreateSnapshot).To(Equal([]time.Duration{timeout}))
		})

		It("provides logging", func() {
			Expect(log).To(gbytes.Say(fmt.Sprintf(`{"event":"starting","task":"create-snapshot","timeout":"%s"}`, timeout.String())))
			Expect(log).To(gbytes.Say(fmt.Sprintf(`{"event":"done","task":"create-snapshot","timeout":"%s"}`, timeout.String())))
			Expect(log).To(gbytes.Say(`{"event":"starting","task":"get-rdb-path"}`))
			Expect(log).To(gbytes.Say(fmt.Sprintf(`{"event":"done","path":"%s","task":"get-rdb-path"}`, expectedArtifactPath)))
		})

		Context("when create snapshot fails", func() {
			var expectedErr = errors.New("create-snapshot-error")

			BeforeEach(func() {
				fakeRedisClient = &fakes.Client{
					ExpectedCreateSnapshotErr: expectedErr,
				}

				logger = lager.NewLogger("logger")
				log = gbytes.NewBuffer()
				logger.RegisterSink(lager.NewWriterSink(log, lager.INFO))

				snapshot := backup.NewSnapshot(fakeRedisClient, 123, logger)
				artifact, err = snapshot.Create()
			})

			It("returns the error", func() {
				Expect(err).To(Equal(expectedErr))
			})

			It("logs the error", func() {
				Expect(log).To(gbytes.Say(fmt.Sprintf(`"error":"%s","event":"failed","task":"create-snapshot"}`, err.Error())))
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

				snapshot := backup.NewSnapshot(fakeRedisClient, 123, logger)
				artifact, err = snapshot.Create()
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
