package backup_test

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/cf-redis-broker/recovery/task"
	"github.com/pivotal-cf/cf-redis-broker/redis/backup"
)

var _ = FDescribe("Cleanup", func() {
	Context(".Name", func() {
		It("returns the name", func() {
			cleanupTask := backup.NewCleanup("dumpRdb", "renamedRdb")

			Expect(cleanupTask.Name()).To(Equal("cleanup"))
		})
	})

	Context(".Run", func() {
		var (
			newDumpFilename     string
			newDumpContents     []byte
			renamedDumpFilename string
			redisDir            string
			err                 error
			cleanup             task.Task
			artifact            task.Artifact = task.NewArtifact("")
		)

		Context("when new dump RDB file exists", func() {
			BeforeEach(func() {
				redisDir, err = ioutil.TempDir("", "cleanup-test")
				Expect(err).ToNot(HaveOccurred())

				newDumpFilename = filepath.Join(redisDir, "dump.rdb")
				renamedDumpFilename = filepath.Join(redisDir, "renamed.rdb")
				newDumpContents = []byte("new dump file")

				err = ioutil.WriteFile(newDumpFilename, newDumpContents, os.ModePerm)
				Expect(err).ToNot(HaveOccurred())
				err = ioutil.WriteFile(renamedDumpFilename, []byte("renamed dump file"), os.ModePerm)
				Expect(err).ToNot(HaveOccurred())

				cleanup = backup.NewCleanup(newDumpFilename, renamedDumpFilename)
			})

			AfterEach(func() {
				os.RemoveAll(redisDir)
			})

			It("should return the artifact", func() {
				returnedArtifact, _ := cleanup.Run(artifact)

				Expect(returnedArtifact).To(Equal(artifact))
			})

			It("should leave the new RDB file as is", func() {
				cleanup.Run(nil)

				contents, err := ioutil.ReadFile(newDumpFilename)
				Expect(err).ToNot(HaveOccurred())

				Expect(contents).To(Equal(newDumpContents))
			})

			It("deletes the renamed RDB file", func() {
				cleanup.Run(nil)

				Expect(renamedDumpFilename).ToNot(BeAnExistingFile())
			})

			Context("when renaming fails", func() {
				It("should return the artifact", func() {
					returnedArtifact, _ := cleanup.Run(artifact)

					Expect(returnedArtifact).To(Equal(artifact))
				})

				It("cant stat dump RDB file then should return the error", func() {
					file, err := os.Open(redisDir)
					Expect(err).ToNot(HaveOccurred())
					file.Chmod(000)

					_, err = cleanup.Run(nil)
					Expect(os.IsPermission(err)).To(BeTrue())
				})
			})

			Context("removing renamed file fails", func() {
				It("should return the artifact", func() {
					returnedArtifact, _ := cleanup.Run(artifact)

					Expect(returnedArtifact).To(Equal(artifact))
				})

				It("renamed RDB file does not exist then it should not return an error", func() {
					os.Remove(renamedDumpFilename)

					_, err = cleanup.Run(nil)
					Expect(err).ToNot(HaveOccurred())
				})

				It("renamed RDB file cannot be accessed then it should return error", func() {
					renamedRdbDir, err := ioutil.TempDir(redisDir, "")
					Expect(err).ToNot(HaveOccurred())
					renamedRdbFile, err := os.Open(renamedRdbDir)
					Expect(err).ToNot(HaveOccurred())
					renamedRdbFile.Chmod(000)

					renamedDumpFilename = path.Join(renamedRdbDir, "renamed_rdb")

					cleanup = backup.NewCleanup(newDumpFilename, renamedDumpFilename)
					_, err = cleanup.Run(nil)
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Context("when no new dump RDB file is present", func() {
			var (
				dumpFilename        string
				renamedDumpFilename string
				renamedDumpContents []byte
			)

			BeforeEach(func() {
				redisDir, err := ioutil.TempDir("", "cleanup-test")
				Expect(err).ToNot(HaveOccurred())

				dumpFilename = filepath.Join(redisDir, "dump.rdb")
				renamedDumpFilename = filepath.Join(redisDir, "renamed.rdb")
				renamedDumpContents = []byte("renamed dump file")

				err = ioutil.WriteFile(renamedDumpFilename, renamedDumpContents, os.ModePerm)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should return the artifact", func() {
				cleanup = backup.NewCleanup(dumpFilename, renamedDumpFilename)
				returnedArtifact, _ := cleanup.Run(artifact)

				Expect(returnedArtifact).To(Equal(artifact))
			})

			It("renames the renamed RDB to original RDB file name", func() {
				cleanup = backup.NewCleanup(dumpFilename, renamedDumpFilename)
				cleanup.Run(nil)

				Expect(renamedDumpFilename).NotTo(BeAnExistingFile())

				contents, err := ioutil.ReadFile(dumpFilename)
				Expect(err).ToNot(HaveOccurred())

				Expect(contents).To(Equal(renamedDumpContents))
			})
		})
	})

})
