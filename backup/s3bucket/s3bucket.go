package s3bucket

import (
	"errors"
	"os"

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

func NewClient(endpointUrl, accessKey, secretKey string) Client {
	auth := aws.Auth{
		AccessKey: accessKey,
		SecretKey: secretKey,
	}
	s3Client := s3.New(auth, getRegion(endpointUrl))
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

func (bucket Bucket) Upload(fileToUpload *os.File, path string) (err error) {
	const partSize = 5242880
	var multi *s3.Multi
	var parts []s3.Part
	if multi, err = bucket.s3Bucket.Multi(path, "", ""); err != nil {
		return err
	}
	if parts, err = multi.PutAll(fileToUpload, partSize); err != nil {
		return err
	}
	return multi.Complete(parts)
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
