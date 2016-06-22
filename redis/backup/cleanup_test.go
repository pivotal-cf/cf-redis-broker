package backup_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-cf/cf-redis-broker/recovery/task"
	"github.com/pivotal-cf/cf-redis-broker/redis/backup"
	"github.com/pivotal-golang/lager"
)

var _ = Describe("Cleanup", func() {
	Context(".Name", func() {
		It("returns the name", func() {
			cleanupTask := backup.NewCleanup("dumpRdb", "renamedRdb", nil)

			Expect(cleanupTask.Name()).To(Equal("cleanup"))
		})
	})

	Context(".Run", func() {
		var (
			originalDumpPath string
			renamedDumpPath  string
			redisDir         string
			cleanup          task.Task
			artifactIn       task.Artifact
			artifactOut      task.Artifact
			runErr           error
			log              *gbytes.Buffer
			logger           lager.Logger
		)

		BeforeEach(func() {
			log = gbytes.NewBuffer()
			logger = lager.NewLogger("redis")
			logger.RegisterSink(lager.NewWriterSink(log, lager.INFO))

			artifactIn = task.NewArtifact("path/to/artifact")

			var err error
			redisDir, err = ioutil.TempDir("", "cleanup-test")
			Expect(err).ToNot(HaveOccurred())

			originalDumpPath = filepath.Join(redisDir, "dump.rdb")
			renamedDumpPath = filepath.Join(redisDir, "renamed.rdb")

			cleanup = backup.NewCleanup(originalDumpPath, renamedDumpPath, logger)
		})

		JustBeforeEach(func() {
			artifactOut, runErr = cleanup.Run(artifactIn)
		})

		AfterEach(func() {
			os.RemoveAll(redisDir)
		})

		Context("when the artifact is nil", func() {
			BeforeEach(func() {
				artifactIn = nil
			})

			It("does not return an error", func() {
				Expect(runErr).ToNot(HaveOccurred())
			})
		})

		Context("when the original dump does not exist", func() {
			Context("when the renamed dump does not exist", func() {
				It("does not return error", func() {
					Expect(runErr).ToNot(HaveOccurred())
				})

				It("returns the passed in artifact", func() {
					Expect(artifactIn).To(Equal(artifactOut))
				})

				It("provides logging", func() {
					Expect(log).To(gbytes.Say(fmt.Sprintf(
						`"redis.cleanup",.*{"event":"starting","original_path":"%s","renamed_path":"%s"}`,
						originalDumpPath,
						renamedDumpPath,
					)))
					Expect(log).To(gbytes.Say(fmt.Sprintf(
						`"redis.cleanup",.*{"event":"done","original_path":"%s","renamed_path":"%s"}`,
						originalDumpPath,
						renamedDumpPath,
					)))
				})
			})

			Context("when the renamed dump exists", func() {
				var expectedContents = []byte("some-content")

				BeforeEach(func() {
					err := ioutil.WriteFile(renamedDumpPath, expectedContents, os.ModePerm)
					Expect(err).ToNot(HaveOccurred())
				})

				It("does not return error", func() {
					Expect(runErr).ToNot(HaveOccurred())
				})

				It("recreates the original dump with the contents from the renamed dump", func() {
					Expect(originalDumpPath).To(BeAnExistingFile())
					contents, err := ioutil.ReadFile(originalDumpPath)
					Expect(err).ToNot(HaveOccurred())
					Expect(contents).To(Equal(expectedContents))
				})

				It("removes the renamed dump", func() {
					Expect(renamedDumpPath).ToNot(BeAnExistingFile())
				})

				It("returns the passed in artifact", func() {
					Expect(artifactOut).To(Equal(artifactIn))
				})

				It("provides logging", func() {
					Expect(log).To(gbytes.Say(fmt.Sprintf(
						`"redis.cleanup",.*"event":"starting","original_path":"%s","renamed_path":"%s"}`,
						originalDumpPath,
						renamedDumpPath,
					)))
					Expect(log).To(gbytes.Say(fmt.Sprintf(
						`"redis.cleanup.rename",.*{"event":"starting","new_path":"%s","old_path":"%s"}`,
						originalDumpPath,
						renamedDumpPath,
					)))
					Expect(log).To(gbytes.Say(fmt.Sprintf(
						`"redis.cleanup.rename",.*{"event":"done","new_path":"%s","old_path":"%s"}`,
						originalDumpPath,
						renamedDumpPath,
					)))
					Expect(log).To(gbytes.Say(fmt.Sprintf(
						`"redis.cleanup",.*"event":"done","original_path":"%s","renamed_path":"%s"}`,
						originalDumpPath,
						renamedDumpPath,
					)))
				})

				Context("when renaming the dump fails", func() {
					var expectedErr = errors.New("rename-error")

					BeforeEach(func() {
						cleanup = backup.NewCleanup(
							originalDumpPath,
							renamedDumpPath,
							logger,
							backup.InjectRenamer(func(string, string) error {
								return expectedErr
							}))
					})

					It("returns the error", func() {
						Expect(runErr).To(Equal(expectedErr))
					})

					It("returns the passed in artifact", func() {
						Expect(artifactOut).To(Equal(artifactIn))
					})

					It("logs the error", func() {
						Expect(log).To(gbytes.Say(fmt.Sprintf(
							`"redis.cleanup.rename",.*{"event":"starting","new_path":"%s","old_path":"%s"}`,
							originalDumpPath,
							renamedDumpPath,
						)))
						Expect(log).To(gbytes.Say(fmt.Sprintf(
							`"redis.cleanup.rename",.*{"error":"%s","event":"failed","new_path":"%s","old_path":"%s"}`,
							expectedErr.Error(),
							originalDumpPath,
							renamedDumpPath,
						)))
					})
				})
			})
		})

		Context("when the original dump exists", func() {
			var expectedContentsOriginalDump = []byte("original-dump")

			BeforeEach(func() {
				err := ioutil.WriteFile(originalDumpPath, expectedContentsOriginalDump, os.ModePerm)
				Expect(err).ToNot(HaveOccurred())
			})

			Context("when the renamed dump does not exist", func() {
				It("does not return error", func() {
					Expect(runErr).ToNot(HaveOccurred())
				})

				It("does not overwrite the original dump", func() {
					actualContents, err := ioutil.ReadFile(originalDumpPath)
					Expect(err).ToNot(HaveOccurred())
					Expect(actualContents).To(Equal(expectedContentsOriginalDump))
				})

				It("returns the passed in artifact", func() {
					Expect(artifactOut).To(Equal(artifactIn))
				})

				It("provides logging", func() {
					Expect(log).To(gbytes.Say(fmt.Sprintf(
						`"redis.cleanup",.*"event":"starting","original_path":"%s","renamed_path":"%s"}`,
						originalDumpPath,
						renamedDumpPath,
					)))
					Expect(log).To(gbytes.Say(fmt.Sprintf(
						`"redis.cleanup",.*"event":"done","original_path":"%s","renamed_path":"%s"}`,
						originalDumpPath,
						renamedDumpPath,
					)))
				})
			})

			Context("when the renamed dump does exist", func() {
				var expectedContentsRenamedDump = []byte("renamed-dump")

				BeforeEach(func() {
					err := ioutil.WriteFile(renamedDumpPath, expectedContentsRenamedDump, os.ModePerm)
					Expect(err).ToNot(HaveOccurred())
				})

				It("does not return error", func() {
					Expect(runErr).ToNot(HaveOccurred())
				})

				It("does not overwrite the original dump", func() {
					actualContents, err := ioutil.ReadFile(originalDumpPath)
					Expect(err).ToNot(HaveOccurred())
					Expect(actualContents).To(Equal(expectedContentsOriginalDump))
				})

				It("deletes the renamed dump", func() {
					Expect(renamedDumpPath).ToNot(BeAnExistingFile())
				})

				It("returns the passed in artifact", func() {
					Expect(artifactOut).To(Equal(artifactIn))
				})

				It("provides logging", func() {
					Expect(log).To(gbytes.Say(fmt.Sprintf(
						`"redis.cleanup",.*"event":"starting","original_path":"%s","renamed_path":"%s"}`,
						originalDumpPath,
						renamedDumpPath,
					)))
					Expect(log).To(gbytes.Say(fmt.Sprintf(
						`"redis.cleanup.remove",.*{"event":"starting","path":"%s"}`,
						renamedDumpPath,
					)))
					Expect(log).To(gbytes.Say(fmt.Sprintf(
						`"redis.cleanup.remove",.*{"event":"done","path":"%s"}`,
						renamedDumpPath,
					)))
					Expect(log).To(gbytes.Say(fmt.Sprintf(
						`"redis.cleanup",.*"event":"done","original_path":"%s","renamed_path":"%s"}`,
						originalDumpPath,
						renamedDumpPath,
					)))
				})

				Context("when renamed dump cannot be removed", func() {
					var expectedErr = errors.New("remove-error")

					BeforeEach(func() {
						cleanup = backup.NewCleanup(
							originalDumpPath,
							renamedDumpPath,
							logger,
							backup.InjectRemover(func(string) error {
								return expectedErr
							}))
					})

					It("returns the error", func() {
						Expect(runErr).To(Equal(expectedErr))
					})

					It("returns the passed in artifact", func() {
						Expect(artifactOut).To(Equal(artifactIn))
					})

					It("logs the error", func() {
						Expect(log).To(gbytes.Say(fmt.Sprintf(
							`"redis.cleanup.remove",.*{"event":"starting","path":"%s"}`,
							renamedDumpPath,
						)))
						Expect(log).To(gbytes.Say(fmt.Sprintf(
							`"redis.cleanup.remove",.*{"error":"%s","event":"failed","path":"%s"}`,
							expectedErr.Error(),
							renamedDumpPath,
						)))
					})
				})
			})
		})
	})
})
