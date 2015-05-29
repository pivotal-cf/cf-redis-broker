package s3

import (
	"github.com/mitchellh/goamz/aws"
	goamz "github.com/mitchellh/goamz/s3"
	"github.com/pivotal-golang/lager"
)

type s3Client struct {
	endpoint    string
	goamzClient *goamz.S3
	logger      lager.Logger
}

type Client interface {
	GetOrCreateBucket(string) (Bucket, error)
}

func NewClient(endpoint, accessKey, secretKey string, logger lager.Logger) Client {
	auth := aws.Auth{
		AccessKey: accessKey,
		SecretKey: secretKey,
	}

	return &s3Client{
		endpoint:    endpoint,
		goamzClient: goamz.New(auth, getRegion(endpoint)),
		logger:      logger,
	}
}

func (c *s3Client) GetOrCreateBucket(bucketName string) (Bucket, error) {
	logData := lager.Data{
		"bucket_name": bucketName,
	}

	c.logInfo("s3client.get-or-create-bucket", "starting", logData)

	bucket := c.goamzClient.Bucket(bucketName)

	switch err := bucket.PutBucket(goamz.Private).(type) {
	case *goamz.Error:
		if err.StatusCode != 409 {
			c.logError("s3client.get-or-create-bucket", err, logData)
			return nil, err
		} else {
			c.logInfo("s3client.get-or-create-bucket", "already-exists", logData)
		}
	case error:
		c.logError("s3client.get-or-create-bucket", err, logData)
		return nil, err
	}

	c.logInfo("s3client.get-or-create-bucket", "done", logData)

	return NewBucket(
		bucket.Name,
		c.endpoint,
		c.goamzClient.AccessKey,
		c.goamzClient.SecretKey,
		c.logger,
	), nil
}

func (c *s3Client) logInfo(action, event string, data lager.Data) {
	data["event"] = event
	c.logger.Info(action, data)
}

func (c *s3Client) logError(action string, err error, data lager.Data) {
	data["event"] = "failed"
	c.logger.Error(action, err, data)
}

func getRegion(endpointUrl string) aws.Region {
	for _, region := range aws.Regions {
		if endpointUrl == region.S3Endpoint {
			return region
		}
	}

	return aws.Region{
		Name:                 "custom-region",
		S3Endpoint:           endpointUrl,
		S3LocationConstraint: true,
	}
}
