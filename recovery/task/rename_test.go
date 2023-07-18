package task_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/lager/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-cf/cf-redis-broker/recovery/task"
)

var _ = Describe("Rename", func() {
	Describe(".Name", func() {
		It("returns name", func() {
			Expect(task.NewRename("blah", nil).Name()).To(Equal("rename"))
		})
	})

	Describe(".Run", func() {
		var (
			tmpDirPath      string
			originalPath    string
			finalPath       string
			renamedArtifact task.Artifact
			runErr          error
			logger          lager.Logger
			log             *gbytes.Buffer
		)

		BeforeEach(func() {
			tmpDirPath, err := ioutil.TempDir("", "temp")
			Expect(err).ToNot(HaveOccurred())

			finalPath = filepath.Join(tmpDirPath, "newName")
			Expect(finalPath).ToNot(BeAnExistingFile())

			file, err := ioutil.TempFile(tmpDirPath, "artifact")
			Expect(err).ToNot(HaveOccurred())

			originalPath = file.Name()
			artifact := task.NewArtifact(originalPath)

			logger = lager.NewLogger("logger")
			log = gbytes.NewBuffer()
			logger.RegisterSink(lager.NewWriterSink(log, lager.INFO))

			renameTask := task.NewRename(finalPath, logger)
			renamedArtifact, runErr = renameTask.Run(artifact)
		})

		AfterEach(func() {
			os.Remove(tmpDirPath)
		})

		It("renames the artifact on disk", func() {
			Expect(originalPath).ToNot(BeAnExistingFile())
			Expect(finalPath).To(BeAnExistingFile())
		})

		It("returns artifact with the new path", func() {
			Expect(renamedArtifact.Path()).To(Equal(finalPath))
		})

		It("does not return error", func() {
			Expect(runErr).ToNot(HaveOccurred())
		})

		It("provides logging", func() {
			Expect(log).To(gbytes.Say(fmt.Sprintf(`{"event":"starting","source":"%s","target":"%s","task":"rename"}`, originalPath, finalPath)))
			Expect(log).To(gbytes.Say(fmt.Sprintf(`{"event":"done","source":"%s","target":"%s","task":"rename"}`, originalPath, finalPath)))
		})

		Context("when an error occurs", func() {
			BeforeEach(func() {
				originalPath = "path/to/nowhere"
				artifact := task.NewArtifact(originalPath)
				renameTask := task.NewRename(finalPath, logger)
				renamedArtifact, runErr = renameTask.Run(artifact)
			})

			It("returns the error", func() {
				Expect(runErr).To(HaveOccurred())
			})

			It("logs the error", func() {
				Expect(log).To(gbytes.Say(fmt.Sprintf(`{"error":"%s","event":"failed","source":"%s","target":"%s","task":"rename"}`, runErr.Error(), originalPath, finalPath)))
			})
		})
	})
})
