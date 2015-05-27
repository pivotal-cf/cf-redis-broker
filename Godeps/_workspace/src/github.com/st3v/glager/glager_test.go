package glager_test

import (
	"errors"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-golang/lager"
	"github.com/st3v/glager"
)

func TestGlager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Glager Test Suite")
}

var _ = Describe("ContainSequence", func() {
	var log *gbytes.Buffer

	Context("when actual is an io.Reader", func() {

		Context("containing a valid lager error log entry", func() {
			var expectedError = errors.New("some-error")

			BeforeEach(func() {
				log = gbytes.NewBuffer()
				logger := lager.NewLogger("logger")
				logger.RegisterSink(lager.NewWriterSink(log, lager.DEBUG))

				logger.Info("action", lager.Data{"event": "starting", "task": "my-task"})
				logger.Debug("action", lager.Data{"event": "debugging", "task": "my-task"})
				logger.Error("action", expectedError, lager.Data{"event": "failed", "task": "my-task"})
			})

			It("matches an info entry", func() {
				Expect(log).To(glager.ContainSequence(
					glager.Info(
						glager.Source("logger"),
						glager.Message("logger.action"),
						glager.Data("event", "starting"),
						glager.Data("task", "my-task"),
					),
				))
			})

			It("matches a debug entry", func() {
				Expect(log).To(glager.ContainSequence(
					glager.Debug(
						glager.Source("logger"),
						glager.Message("logger.action"),
						glager.Data("event", "debugging"),
						glager.Data("task", "my-task"),
					),
				))
			})

			It("matches an error entry", func() {
				Expect(log).To(glager.ContainSequence(
					glager.Error(
						errors.New("some-error"),
						glager.Source("logger"),
						glager.Message("logger.action"),
						glager.Data("event", "failed"),
						glager.Data("task", "my-task"),
					),
				))
			})

			It("does match a correct sequence", func() {
				Expect(log).To(glager.ContainSequence(
					glager.Info(
						glager.Data("event", "starting", "task", "my-task"),
					),
					glager.Debug(
						glager.Data("event", "debugging", "task", "my-task"),
					),
					glager.Error(
						errors.New("some-error"),
						glager.Data("event", "failed", "task", "my-task"),
					),
				))
			})

			It("does not match an incorrect sequence", func() {
				Expect(log).ToNot(glager.ContainSequence(
					glager.Info(
						glager.Data("event", "starting", "task", "my-task"),
					),
					glager.Info(
						glager.Data("event", "starting", "task", "my-task"),
					),
				))
			})

			It("does not match an out-of-order sequence", func() {
				Expect(log).ToNot(glager.ContainSequence(
					glager.Debug(
						glager.Data("event", "debugging", "task", "my-task"),
					),
					glager.Error(
						errors.New("some-error"),
						glager.Data("event", "failed", "task", "my-task"),
					),
					glager.Info(
						glager.Data("event", "starting", "task", "my-task"),
					),
				))
			})

			It("does not match a fatal entry", func() {
				Expect(log).ToNot(glager.ContainSequence(
					glager.Fatal(
						glager.Source("logger"),
						glager.Data("event", "failed", "task", "my-task"),
					),
				))
			})
		})
	})
})
