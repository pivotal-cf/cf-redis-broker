package s3_test

import (
	"errors"
	"fmt"
	"os"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/pivotal-cf/cf-redis-broker/s3"
	"github.com/pivotal-golang/lager"
)

type fakeCommand struct {
	Output                string
	CombinedOutputErr     error
	CombinedOutputInvoked int
}

func (cmd *fakeCommand) CombinedOutput() ([]byte, error) {
	cmd.CombinedOutputInvoked++
	return []byte(cmd.Output), cmd.CombinedOutputErr
}

var _ = Describe("Bucket", func() {
	Describe(".Name", func() {
		It("returns the assigned name", func() {
			bucket := s3.NewBucket("bucket-name", "endpoint", "key", "secret", nil)
			Expect(bucket.Name()).To(Equal("bucket-name"))
		})
	})

	Describe(".Upload", func() {
		var (
			targetPath         = "path/to/target"
			bucketName         = "my-bucket"
			expectedSourcePath = "path/to/source"
			expectedEndpoint   = "http://foo.bar"
			expectedEnvKey     = "FOO"
			expectedEnvVar     = "BAR"
			expectedKey        = "AWS-ACCESS-KEY"
			expectedSecret     = "AWS-SECRET-KEY"
			expectedBucketPath string
			uploadErr          error
			bucket             s3.Bucket
			cmd                *fakeCommand
			cmdEnvironment     []string
			cmdFactory         s3.CommandFactory
			generatedCmd       string
			logger             lager.Logger
			log                *gbytes.Buffer
		)

		BeforeEach(func() {
			expectedBucketPath = fmt.Sprintf("s3://%s/%s", bucketName, targetPath)

			cmd = &fakeCommand{}

			generatedCmd = ""
			cmdFactory = func(name string, env []string, args ...string) s3.Command {
				generatedCmd = fmt.Sprintf("%s %s", name, strings.Join(args, " "))
				cmdEnvironment = env
				return cmd
			}

			logger = lager.NewLogger("logger")
			log = gbytes.NewBuffer()
			logger.RegisterSink(lager.NewWriterSink(log, lager.INFO))

			bucket = s3.NewBucket(
				bucketName,
				expectedEndpoint,
				expectedKey,
				expectedSecret,
				logger,
				s3.InjectCommandFactory(cmdFactory),
			)

			err := os.Setenv(expectedEnvKey, expectedEnvVar)
			Expect(err).ToNot(HaveOccurred())
		})

		JustBeforeEach(func() {
			uploadErr = bucket.Upload(expectedSourcePath, targetPath)
		})

		It("creates a command with the right arguments", func() {
			expectedCommand := fmt.Sprintf(
				"aws s3 cp %s %s --endpoint-url %s",
				expectedSourcePath,
				expectedBucketPath,
				expectedEndpoint,
			)
			Expect(generatedCmd).To(Equal(expectedCommand))
		})

		It("executes the generated command", func() {
			Expect(cmd.CombinedOutputInvoked).To(Equal(1))
		})

		It("does not return an error", func() {
			Expect(uploadErr).ToNot(HaveOccurred())
		})

		It("shells out with the current process environment", func() {
			Expect(cmdEnvironment).To(ContainElement(
				fmt.Sprintf("%s=%s", expectedEnvKey, expectedEnvVar),
			))
		})

		It("shells out with the right env variable for the access key", func() {
			Expect(cmdEnvironment).To(ContainElement(
				fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", expectedKey),
			))
		})

		It("shells out with the right env variable for the secret key", func() {
			Expect(cmdEnvironment).To(ContainElement(
				fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", expectedSecret),
			))
		})

		It("provides logging", func() {
			logFormat := fmt.Sprintf(
				`{"aws_access_key":"%s","aws_secret_key":"%s","bucket_path":"%s","cli_path":"%s","endpoint":"%s","event":"%s","source_path":"%s"}`,
				expectedKey,
				"\\*{11}KEY",
				expectedBucketPath,
				"aws",
				expectedEndpoint,
				"%s",
				expectedSourcePath,
			)

			Expect(log).To(gbytes.Say(fmt.Sprintf(logFormat, "creating-command")))
			Expect(log).To(gbytes.Say(fmt.Sprintf(logFormat, "shelling-out")))
			Expect(log).To(gbytes.Say(fmt.Sprintf(logFormat, "done")))
		})

		Context("when the caller sets a custom aws cli path", func() {
			var expectedAwsCliPath = "path/to/aws/cli"

			BeforeEach(func() {
				bucket = s3.NewBucket(
					bucketName,
					expectedEndpoint,
					expectedKey,
					expectedSecret,
					logger,
					s3.AwsCliPath(expectedAwsCliPath),
					s3.InjectCommandFactory(cmdFactory),
				)
			})

			It("allows for custom aws cli paths", func() {
				args := strings.Split(generatedCmd, " ")
				Expect(args[0]).To(Equal(expectedAwsCliPath))
			})
		})

		Context("when upload command fails", func() {
			var expectedErr = errors.New("some-cmd-error")

			BeforeEach(func() {
				cmd = &fakeCommand{
					CombinedOutputErr: expectedErr,
				}

				generatedCmd = ""
				cmdFactory = func(name string, env []string, args ...string) s3.Command {
					generatedCmd = fmt.Sprintf("%s %s", name, strings.Join(args, " "))
					cmdEnvironment = env
					return cmd
				}

				bucket = s3.NewBucket(
					bucketName,
					expectedEndpoint,
					expectedKey,
					expectedSecret,
					logger,
					s3.InjectCommandFactory(cmdFactory),
				)
			})

			It("returns the error", func() {
				Expect(uploadErr).To(Equal(expectedErr))
			})

			It("logs the error", func() {
				Expect(log).To(gbytes.Say(fmt.Sprintf(
					`{"aws_access_key":"%s","aws_secret_key":"%s","bucket_path":"%s","cli_output":"","cli_path":"%s","endpoint":"%s","error":"%s","event":"%s","source_path":"%s"}`,
					expectedKey,
					"\\*{11}KEY",
					expectedBucketPath,
					"aws",
					expectedEndpoint,
					expectedErr.Error(),
					"failed",
					expectedSourcePath,
				)))
			})
		})
	})
})
