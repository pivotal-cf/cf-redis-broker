package task

import (
	"fmt"

	"github.com/pivotal-cf/cf-redis-broker/recovery"
)

type s3upload struct {
	bucket   string
	endpoint string
	key      string
	secret   string
}

func NewS3Upload(bucket, endpoint, key, secret string) recovery.Task {
	return &s3upload{
		bucket:   bucket,
		endpoint: endpoint,
		key:      key,
		secret:   secret,
	}
}

func (u *s3upload) Run(artifact recovery.Artifact) (recovery.Artifact, error) {
	fmt.Printf("S3 Upload of artifact %s\n", artifact.Path())
	return artifact, nil
}

func (u *s3upload) Name() string {
	return "s3upload"
}
