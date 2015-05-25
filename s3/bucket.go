package s3

import (
	"fmt"
	"math"
	"os"
	"os/exec"

	"github.com/pivotal-golang/lager"
)

type Bucket interface {
	Upload(source, destination string) error
	Name() string
}

type s3Bucket struct {
	awsCliPath string
	name       string
	endpoint   string
	key        string
	secret     string
	logger     lager.Logger
	cmdFactory CommandFactory
}

type Command interface {
	CombinedOutput() ([]byte, error)
}

type CommandFactory func(name string, env []string, arg ...string) Command

func InjectCommandFactory(factory CommandFactory) option {
	return func(b *s3Bucket) {
		b.cmdFactory = factory
	}
}

func AwsCliPath(path string) option {
	return func(b *s3Bucket) {
		b.awsCliPath = path
	}
}

type option func(*s3Bucket)

func NewBucket(name, endpoint, key, secret string, logger lager.Logger, options ...option) *s3Bucket {
	defaultCmdFactory := func(name string, env []string, args ...string) Command {
		cmd := exec.Command(name, args...)
		cmd.Env = env
		return cmd
	}

	bucket := &s3Bucket{
		name:       name,
		endpoint:   endpoint,
		key:        key,
		secret:     secret,
		logger:     logger,
		cmdFactory: defaultCmdFactory,
		awsCliPath: "aws",
	}

	for _, option := range options {
		option(bucket)
	}

	return bucket
}

func (b *s3Bucket) Upload(source, target string) error {
	bucketPath := fmt.Sprintf("s3://%s/%s", b.name, target)

	env := append(
		os.Environ(),
		fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", b.key),
		fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", b.secret),
	)

	logData := lager.Data{
		"cli_path":       b.awsCliPath,
		"source_path":    source,
		"bucket_path":    bucketPath,
		"endpoint":       b.endpoint,
		"aws_access_key": b.key,
		"aws_secret_key": obfuscate(b.secret),
	}

	b.logInfo("s3bucket.upload", "creating-command", logData)

	cmd := b.cmdFactory(
		b.awsCliPath,
		env,
		"s3",
		"cp",
		source,
		bucketPath,
		"--endpoint-url",
		b.endpoint,
	)

	b.logInfo("s3bucket.upload", "shelling-out", logData)

	if _, err := cmd.CombinedOutput(); err != nil {
		b.logError("s3bucket.upload", err, logData)
		return err
	}

	b.logInfo("s3bucket.upload", "done", logData)

	return nil
}

func (b *s3Bucket) Name() string {
	return b.name
}

func (b *s3Bucket) logInfo(action, event string, data lager.Data) {
	data["event"] = event
	b.logger.Info(action, data)
}

func (b *s3Bucket) logError(action string, err error, data lager.Data) {
	data["event"] = "failed"
	b.logger.Error(action, err, data)
}

func obfuscate(txt string) string {
	runes := make([]rune, len(txt))

	cutoff := len(txt) - int(math.Min(float64(len(txt))/4.0, 4.0))

	for i, c := range txt {
		if cutoff > 0 && i >= cutoff {
			runes[i] = c
		} else {
			runes[i] = '*'
		}
	}

	return string(runes)
}
