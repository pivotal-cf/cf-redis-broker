package integration_test

import (
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/goamz/goamz/aws"
	goamz "github.com/goamz/goamz/s3"
	"github.com/goamz/goamz/s3/s3test"

	"github.com/pivotal-cf/cf-redis-broker/integration"
	"github.com/pivotal-cf/cf-redis-broker/redis/backup"
	redis "github.com/pivotal-cf/cf-redis-broker/redis/client"
	"github.com/pivotal-golang/lager"
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Backup Integration Suite")
}

var (
	s3Server         *s3test.Server
	redisRunner      *integration.RedisRunner
	backupErr        error
	uploadedObject   []byte
	localDumpRdbPath string
)

var _ = BeforeSuite(func() {
	redisHost := "127.0.0.1"
	redisPort := integration.RedisPort

	redisRunner = &integration.RedisRunner{}
	redisRunner.Start([]string{"--bind", redisHost, "--port", fmt.Sprintf("%d", redisPort)})

	localDumpRdbPath = filepath.Join(redisRunner.Dir, "dump.rdb")

	var err error
	s3Server, err = s3test.NewServer(&s3test.Config{
		Send409Conflict: true,
	})
	Expect(err).ToNot(HaveOccurred())

	logger := lager.NewLogger("logger")

	client, err := redis.Connect(redis.Host(redisHost), redis.Port(redisPort))
	Expect(err).ToNot(HaveOccurred())

	Expect(localDumpRdbPath).ToNot(BeAnExistingFile())

	bucketName := "test-bucket"
	targetPath := "target"

	bucket := goamzBucket(bucketName, s3Server.URL())
	_, err = bucket.Get(targetPath)
	Expect(err).To(HaveOccurred())

	backuper := backup.NewRedisBackuper(
		10*time.Second,
		bucketName,
		s3Server.URL(),
		"access-key",
		"secret-key",
		logger,
	)

	backupErr = backuper.Backup(
		client,
		targetPath,
	)

	uploadedObject, err = bucket.Get(targetPath)
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	redisRunner.Stop()
	s3Server.Quit()
})

var _ = Describe("backup.Backup", func() {
	It("returns no error", func() {
		Expect(backupErr).ToNot(HaveOccurred())
	})

	It("uploads the artifact to s3", func() {
		Expect(uploadedObject).ToNot(BeEmpty())
	})

	It("leaves us with a dump.rdb", func() {
		Expect(localDumpRdbPath).To(BeAnExistingFile())
	})

	It("uploaded file is the same as the local dump.rdb", func() {
		dumpData, err := ioutil.ReadFile(localDumpRdbPath)
		Expect(err).ToNot(HaveOccurred())
		Expect(md5.Sum(dumpData)).To(Equal(md5.Sum(uploadedObject)))
	})
})

func goamzBucket(bucketName, endpoint string) *goamz.Bucket {
	region := aws.Region{
		Name:                 "fake_region",
		S3Endpoint:           s3Server.URL(),
		S3LocationConstraint: true,
	}
	return goamz.New(aws.Auth{}, region).Bucket(bucketName)
}
