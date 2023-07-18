package task

import (
	"fmt"

	"code.cloudfoundry.org/lager/v3"
	"github.com/pivotal-cf/cf-redis-broker/s3"
)

type s3upload struct {
	client     s3.Client
	bucketName string
	targetPath string
	endpoint   string
	key        string
	secret     string
	logger     lager.Logger
}

func InjectS3Client(s3client s3.Client) S3UploadInjector {
	return func(u *s3upload) {
		u.client = s3client
	}
}

type S3UploadInjector func(*s3upload)

func NewS3Upload(
	bucketName, targetPath, endpoint, key, secret string,
	logger lager.Logger,
	injectors ...S3UploadInjector,
) Task {
	upload := &s3upload{
		bucketName: bucketName,
		targetPath: targetPath,
		endpoint:   endpoint,
		key:        key,
		secret:     secret,
		client:     s3.NewClient(endpoint, key, secret, logger),
		logger:     logger,
	}

	for _, injector := range injectors {
		injector(upload)
	}

	return upload
}

func (u *s3upload) Run(artifact Artifact) (Artifact, error) {
	logData := lager.Data{
		"source_path": artifact.Path(),
		"target_path": u.targetPath,
		"bucket":      u.bucketName,
	}
	u.logInfo("", "starting", logData)

	bucket, err := u.createBucket()
	if err != nil {
		u.logError("", err, logData)
		return nil, err
	}

	err = u.uploadToBucket(bucket, artifact.Path())
	if err != nil {
		u.logError("", err, logData)
		return nil, err
	}

	u.logInfo("", "done", logData)

	return artifact, nil
}

func (u *s3upload) Name() string {
	return "s3upload"
}

func (u *s3upload) createBucket() (s3.Bucket, error) {
	logData := lager.Data{
		"bucket": u.bucketName,
	}

	u.logInfo("create-bucket", "starting", logData)

	bucket, err := u.client.GetOrCreateBucket(u.bucketName)
	if err != nil {
		u.logError("create-bucket", err, logData)
		return nil, err
	}

	u.logInfo("create-bucket", "done", logData)

	return bucket, nil
}

func (u *s3upload) uploadToBucket(bucket s3.Bucket, sourcePath string) error {
	logData := lager.Data{
		"source_path": sourcePath,
		"target_path": u.targetPath,
		"bucket":      u.bucketName,
	}

	u.logInfo("upload", "starting", logData)

	err := bucket.Upload(sourcePath, u.targetPath)
	if err != nil {
		u.logError("upload", err, logData)
		return err
	}

	u.logInfo("upload", "done", logData)

	return nil
}

func (u *s3upload) logError(subAction string, err error, data lager.Data) {
	data["event"] = "failed"
	u.logger.Error(u.logAction(subAction), err, data)
}

func (u *s3upload) logInfo(subAction, event string, data lager.Data) {
	data["event"] = event

	u.logger.Info(
		u.logAction(subAction),
		data,
	)
}

func (u *s3upload) logAction(subAction string) string {
	action := u.Name()
	if subAction != "" {
		action = fmt.Sprintf("%s.%s", action, subAction)
	}

	return action
}
