package s3bucket

import (
	"errors"

	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
)

type Client struct {
	s3Client *s3.S3
}

type Bucket struct {
	Name string

	s3Bucket *s3.Bucket
}

var S3RegionNotFoundErr error = errors.New("S3 region not found")

func NewClient(endpointUrl, regionName, accessKey, secretKey string) Client {
	region := aws.Region{
		Name:                 regionName,
		S3Endpoint:           endpointUrl,
		S3LocationConstraint: true,
	}

	auth := aws.Auth{
		AccessKey: accessKey,
		SecretKey: secretKey,
	}

	s3Client := s3.New(auth, region)
	return Client{
		s3Client: s3Client,
	}
}

func (client Client) GetOrCreate(bucketName string) (Bucket, error) {
	s3Bucket := client.s3Client.Bucket(bucketName)

	switch err := s3Bucket.PutBucket(s3.Private).(type) {
	case *s3.Error:
		if err.StatusCode != 409 {
			return Bucket{}, err
		}
	case error:
		return Bucket{}, err
	}

	return Bucket{
		Name:     s3Bucket.Name,
		s3Bucket: s3Bucket,
	}, nil
}

func (bucket Bucket) Upload(data []byte, path string) error {
	return bucket.s3Bucket.Put(path, data, "", "")
}
