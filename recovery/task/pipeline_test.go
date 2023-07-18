package task_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	glager "github.com/st3v/glager"

	"code.cloudfoundry.org/lager/v3"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-cf/cf-redis-broker/recovery/task"
)

type fakeTask struct {
	ExpectedErr error
	Artifact    task.Artifact
	TaskName    string
}

func (f *fakeTask) Name() string {
	return f.TaskName
}

func (f *fakeTask) Run(artifact task.Artifact) (task.Artifact, error) {
	f.Artifact = artifact
	return task.NewArtifact(f.Name()), f.ExpectedErr
}

var _ = Describe("Pipeline", func() {
	var (
		logger lager.Logger
		log    *gbytes.Buffer
	)

	BeforeEach(func() {
		logger = lager.NewLogger("logger")
		log = gbytes.NewBuffer()
		logger.RegisterSink(lager.NewWriterSink(log, lager.INFO))
	})

	Describe(".Name", func() {
		It("returns the pipeline name", func() {
			pipeline := task.NewPipeline("some-pipeline", logger)

			Expect(pipeline.Name()).To(Equal("some-pipeline"))
		})

	})

	Describe(".Run", func() {
		var (
			task1            *fakeTask
			task2            *fakeTask
			originalArtifact = task.NewArtifact("path/to/artifact")
			finalArtifact    task.Artifact
			runErr           error
		)

		Context("with multiple successful tasks", func() {
			BeforeEach(func() {
				task1 = &fakeTask{TaskName: "task1"}
				task2 = &fakeTask{TaskName: "task2"}

				pipeline := task.NewPipeline("some-name", logger, task1, task2)

				finalArtifact, runErr = pipeline.Run(originalArtifact)
			})

			It("executes all tasks with given artifact", func() {
				Expect(task1.Artifact).To(Equal(originalArtifact))
				Expect(task2.Artifact.Path()).To(Equal("task1"))
			})

			It("returns the artifact of the last task", func() {
				Expect(finalArtifact.Path()).To(Equal("task2"))
			})

			It("logs each step", func() {
				Expect(log).To(glager.ContainSequence(
					glager.Info(glager.Data("event", "starting", "pipeline", "some-name", "task", "task1")),
					glager.Info(glager.Data("event", "done", "pipeline", "some-name", "task", "task1")),
					glager.Info(glager.Data("event", "starting", "pipeline", "some-name", "task", "task2")),
					glager.Info(glager.Data("event", "done", "pipeline", "some-name", "task", "task2")),
				))
			})

			It("does not return an error", func() {
				Expect(runErr).ToNot(HaveOccurred())
			})
		})

		Context("when one of the tasks fails", func() {
			var (
				task3         *fakeTask
				expectedError = errors.New("some-task-error")
			)

			BeforeEach(func() {
				task1 = &fakeTask{TaskName: "task1"}
				task2 = &fakeTask{TaskName: "task2", ExpectedErr: expectedError}
				task3 = &fakeTask{TaskName: "task3"}

				pipeline := task.NewPipeline("some-name", logger, task1, task2, task3)

				finalArtifact, runErr = pipeline.Run(originalArtifact)
			})

			It("short circuits the tasks chain", func() {
				Expect(task1.Artifact).ToNot(BeNil())
				Expect(task2.Artifact).ToNot(BeNil())
				Expect(task3.Artifact).To(BeNil())
			})

			It("returns the error", func() {
				Expect(runErr).To(Equal(expectedError))
			})

			It("logs the error", func() {
				Expect(log).To(glager.ContainSequence(
					glager.Info(glager.Data("event", "starting", "pipeline", "some-name", "task", "task1")),
					glager.Info(glager.Data("event", "done", "pipeline", "some-name", "task", "task1")),
					glager.Info(glager.Data("event", "starting", "pipeline", "some-name", "task", "task2")),
					glager.Error(expectedError, glager.Data("event", "failed", "pipeline", "some-name", "task", "task2")),
				))
			})
		})
	})
})
