package glager_test

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"
	. "github.com/st3v/glager"
)

func TestGlager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Glager Test Suite")
}

var _ = Describe(".ContainSequence", func() {
	var (
		buffer         *gbytes.Buffer
		logger         lager.Logger
		expectedSource = "some-source"
	)

	BeforeEach(func() {
		buffer = gbytes.NewBuffer()
		logger = lager.NewLogger(expectedSource)
		logger.RegisterSink(lager.NewWriterSink(buffer, lager.DEBUG))
	})

	Context("when actual contains an entry", func() {
		var (
			action            = "some-action"
			expectedAction    = fmt.Sprintf("%s.%s", expectedSource, action)
			expectedDataKey   = "some-key"
			expectedDataValue = "some-value"
		)

		Context("that is an info", func() {
			BeforeEach(func() {
				logger.Info(action, lager.Data{expectedDataKey: expectedDataValue})
			})

			It("matches an empty info entry", func() {
				Expect(buffer).To(ContainSequence(
					Info(),
				))
			})

			It("matches an info entry with a source only", func() {
				Expect(buffer).To(ContainSequence(
					Info(
						Source(expectedSource),
					),
				))
			})

			It("matches an info entry with a message only", func() {
				Expect(buffer).To(ContainSequence(
					Info(
						Message(expectedAction),
					),
				))
			})

			It("matches an info entry with an action only", func() {
				Expect(buffer).To(ContainSequence(
					Info(
						Action(expectedAction),
					),
				))
			})

			It("matches an info entry with data only", func() {
				Expect(buffer).To(ContainSequence(
					Info(
						Data(expectedDataKey, expectedDataValue),
					),
				))
			})

			It("matches the correct info entry", func() {
				Expect(buffer).To(ContainSequence(
					Info(
						Source(expectedSource),
						Message(expectedAction),
						Data(expectedDataKey, expectedDataValue),
					),
				))
			})

			It("does not match an info entry with an incorrect source", func() {
				Expect(buffer).ToNot(ContainSequence(
					Info(
						Source("invalid"),
						Message(expectedAction),
						Data(expectedDataKey, expectedDataValue),
					),
				))
			})

			It("does not match an info entry with an incorrect message", func() {
				Expect(buffer).ToNot(ContainSequence(
					Info(
						Source(expectedSource),
						Message("invalid"),
						Data(expectedDataKey, expectedDataValue),
					),
				))
			})

			It("does not match an info entry with incorrect data", func() {
				Expect(buffer).ToNot(ContainSequence(
					Info(
						Source(expectedSource),
						Message(expectedAction),
						Data(expectedDataKey, expectedDataValue, "non-existing-key", "non-existing-value"),
					),
				))
			})

			It("does not match a debug entry", func() {
				Expect(buffer).ToNot(ContainSequence(Debug()))
			})

			It("does not match an error entry", func() {
				Expect(buffer).ToNot(ContainSequence(Error(nil)))
			})

			It("does not match a fatal entry", func() {
				Expect(buffer).ToNot(ContainSequence(Fatal(nil)))
			})
		})

		Context("that is an error", func() {
			var expectedErr = errors.New("some-error")

			BeforeEach(func() {
				logger.Error(action, expectedErr, lager.Data{expectedDataKey: expectedDataValue})
			})

			It("does match the correct error without additional fields", func() {
				Expect(buffer).To(ContainSequence(
					Error(
						expectedErr,
					),
				))
			})

			It("does match the correct error with correct additional fields", func() {
				Expect(buffer).To(ContainSequence(
					Error(
						expectedErr,
						Source(expectedSource),
						Action(expectedAction),
						Data(expectedDataKey, expectedDataValue),
					),
				))
			})

			It("does not match an incorrect error", func() {
				Expect(buffer).ToNot(ContainSequence(Error(errors.New("some-other-error"))))
			})

			It("does not match the correct error with incorrect source", func() {
				Expect(buffer).ToNot(ContainSequence(
					Error(
						expectedErr,
						Source("incorrect"),
					),
				))
			})

			It("does not match the correct error with incorrect message", func() {
				Expect(buffer).ToNot(ContainSequence(
					Error(
						expectedErr,
						Message("incorrect"),
					),
				))
			})

			It("does not match the correct error with incorrect data", func() {
				Expect(buffer).ToNot(ContainSequence(
					Error(
						expectedErr,
						Data("non-exiting-key", "non-existing-value"),
					),
				))
			})

			It("does not match an info entry", func() {
				Expect(buffer).ToNot(ContainSequence(Info()))
			})

			It("does not match a debug entry", func() {
				Expect(buffer).ToNot(ContainSequence(Debug()))
			})

			It("does not match a fatal entry", func() {
				Expect(buffer).ToNot(ContainSequence(Fatal(nil)))
			})
		})

		Context("that is a debug entry", func() {
			BeforeEach(func() {
				logger.Debug(action, lager.Data{expectedDataKey: expectedDataValue})
			})

			It("does match an empty debug entry", func() {
				Expect(buffer).To(ContainSequence(Debug()))
			})

			It("does match the correct debug entry", func() {
				Expect(buffer).To(ContainSequence(
					Debug(
						Source(expectedSource),
						Message(expectedAction),
						Data(expectedDataKey, expectedDataValue),
					),
				))
			})

			It("does not match a debug entry with an incorrect source", func() {
				Expect(buffer).ToNot(ContainSequence(
					Debug(
						Source("incorrect"),
					),
				))
			})

			It("does not match a debug entry with an incorrect message", func() {
				Expect(buffer).ToNot(ContainSequence(
					Debug(
						Message("incorrect"),
					),
				))
			})

			It("does not match a debug entry with a incorrect data", func() {
				Expect(buffer).ToNot(ContainSequence(
					Debug(
						Data("non-existing-key"),
					),
				))
			})

			It("does not match an info entry", func() {
				Expect(buffer).ToNot(ContainSequence(Info()))
			})

			It("does not match an error entry", func() {
				Expect(buffer).ToNot(ContainSequence(Error(nil)))
			})

			It("does not match a fatal entry", func() {
				Expect(buffer).ToNot(ContainSequence(Fatal(nil)))
			})
		})

		Context("that is a fatal error", func() {
			var expectedErr = errors.New("some-error")

			BeforeEach(func() {
				func() {
					defer func() {
						recover()
					}()

					logger.Fatal(action, expectedErr, lager.Data{expectedDataKey: expectedDataValue})
				}()
			})

			It("does match an empty fatal entry", func() {
				Expect(buffer).To(ContainSequence(Fatal(nil)))
			})

			It("does match a fatal entry with correct error", func() {
				Expect(buffer).To(ContainSequence(
					Fatal(
						expectedErr,
					),
				))
			})

			It("does match a fatal entry with correct error and additional fields", func() {
				Expect(buffer).To(ContainSequence(
					Fatal(
						expectedErr,
						Source(expectedSource),
						Message(expectedAction),
						Data(expectedDataKey, expectedDataValue),
					),
				))
			})

			It("does not match a fatal entry with an incorrect error", func() {
				Expect(buffer).ToNot(ContainSequence(
					Fatal(
						errors.New("some-other-error"),
					),
				))
			})

			It("does not match a fatal entry with an incorrect source", func() {
				Expect(buffer).ToNot(ContainSequence(
					Fatal(
						expectedErr,
						Source("incorrect"),
					),
				))
			})

			It("does not match a fatal entry with an incorrect action", func() {
				Expect(buffer).ToNot(ContainSequence(
					Fatal(
						expectedErr,
						Action("incorrect"),
					),
				))
			})

			It("does not match a fatal entry with incorrect data", func() {
				Expect(buffer).ToNot(ContainSequence(
					Fatal(
						expectedErr,
						Data("incorrect"),
					),
				))
			})

			It("does not match an info entry", func() {
				Expect(buffer).ToNot(ContainSequence(Info()))
			})

			It("does not match a debug entry", func() {
				Expect(buffer).ToNot(ContainSequence(Debug()))
			})

			It("does not match an error entry", func() {
				Expect(buffer).ToNot(ContainSequence(Error(nil)))
			})
		})
	})

	Context("when actual contains multiple entries", func() {
		var expectedError = errors.New("some-error")

		BeforeEach(func() {
			logger.Info("action", lager.Data{"event": "starting", "task": "my-task"})
			logger.Debug("action", lager.Data{"event": "debugging", "task": "my-task"})
			logger.Error("action", expectedError, lager.Data{"event": "failed", "task": "my-task"})
		})

		It("does match a correct sequence", func() {
			Expect(buffer).To(ContainSequence(
				Info(
					Data("event", "starting", "task", "my-task"),
				),
				Debug(
					Data("event", "debugging", "task", "my-task"),
				),
				Error(
					expectedError,
					Data("event", "failed", "task", "my-task"),
				),
			))
		})

		It("does match a correct subsequence with missing elements in the beginning", func() {
			Expect(buffer).To(ContainSequence(
				Debug(
					Data("event", "debugging", "task", "my-task"),
				),
				Error(
					expectedError,
					Data("event", "failed", "task", "my-task"),
				),
			))
		})

		It("does match a correct subsequence with missing elements in the end", func() {
			Expect(buffer).To(ContainSequence(
				Info(
					Data("event", "starting", "task", "my-task"),
				),
				Debug(
					Data("event", "debugging", "task", "my-task"),
				),
			))
		})

		It("does match a correct but non-continious subsequence", func() {
			Expect(buffer).To(ContainSequence(
				Info(
					Data("event", "starting", "task", "my-task"),
				),
				Error(
					expectedError,
					Data("event", "failed", "task", "my-task"),
				),
			))
		})

		It("does not match an incorrect sequence", func() {
			Expect(buffer).ToNot(ContainSequence(
				Info(
					Data("event", "starting", "task", "my-task"),
				),
				Info(
					Data("event", "starting", "task", "my-task"),
				),
			))
		})

		It("does not match an out-of-order sequence", func() {
			Expect(buffer).ToNot(ContainSequence(
				Debug(
					Data("event", "debugging", "task", "my-task"),
				),
				Error(
					expectedError,
					Data("event", "failed", "task", "my-task"),
				),
				Info(
					Data("event", "starting", "task", "my-task"),
				),
			))
		})
	})

	Describe("logMatcher", func() {
		var (
			buffer  *gbytes.Buffer
			logger  lager.Logger
			matcher = ContainSequence(Info())
		)

		BeforeEach(func() {
			buffer = gbytes.NewBuffer()
			logger = lager.NewLogger("logger")
			logger.RegisterSink(lager.NewWriterSink(buffer, lager.DEBUG))
			logger.Debug("some-debug")
		})

		Describe("Match", func() {
			var (
				actual  interface{}
				success bool
				err     error
			)

			JustBeforeEach(func() {
				logger.Info("some-info")
				success, err = matcher.Match(actual)
			})

			Context("when actual is an invalid type", func() {
				BeforeEach(func() {
					actual = "invalid"
				})

				It("returns failure", func() {
					Expect(success).To(BeFalse())
				})

				It("returns an error", func() {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("ContainSequence must be passed"))
				})
			})

			Context("when actual is a BufferProvider", func() {
				var sink *lagertest.TestSink

				BeforeEach(func() {
					sink = lagertest.NewTestSink()
					logger.RegisterSink(sink)
					actual = sink
				})

				It("returns success", func() {
					Expect(success).To(BeTrue())
				})

				It("does not return an error", func() {
					Expect(err).ToNot(HaveOccurred())
				})

				It("does match on subsequent calls", func() {
					Expect(actual).To(matcher)
				})
			})

			Context("when actual is a ContentsProvider", func() {
				BeforeEach(func() {
					actual = buffer
				})

				It("returns success", func() {
					Expect(success).To(BeTrue())
				})

				It("does not return an error", func() {
					Expect(err).ToNot(HaveOccurred())
				})

				It("does match on subsequent calls", func() {
					Expect(actual).To(matcher)
				})
			})

			Context("when actual is an io.Reader", func() {
				BeforeEach(func() {
					actual = bufio.NewReader(buffer)
				})

				It("returns success", func() {
					Expect(success).To(BeTrue())
				})

				It("does not return an error", func() {
					Expect(err).ToNot(HaveOccurred())
				})

				It("does not match on subsequent calls", func() {
					Expect(actual).ToNot(matcher)
				})
			})

			Context("when actual contains invalid entries", func() {
				BeforeEach(func() {
					actual = strings.NewReader("invalid")
				})

				It("returns failure", func() {
					Expect(success).To(BeFalse())
				})

				It("returns a json.SyntaxError", func() {
					Expect(err).To(HaveOccurred())
					Expect(err).To(BeAssignableToTypeOf(&json.SyntaxError{}))
				})
			})

		})

		Describe("FailureMessage", func() {
			It("returns the right message", func() {
				matcher.Match(buffer)
				Expect(matcher.FailureMessage(buffer)).To(ContainSubstring(
					"to contain log sequence",
				))
			})
		})

		Describe("NegatedFailureMessage", func() {
			It("returns the right message", func() {
				matcher.Match(buffer)
				Expect(matcher.NegatedFailureMessage(buffer)).To(ContainSubstring(
					"not to contain log sequence",
				))
			})
		})
	})
})
