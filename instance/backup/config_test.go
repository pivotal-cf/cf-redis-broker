package backup_test

import (
	"path"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/cf-redis-broker/instance/backup"
)

var _ = Describe("LoadBackupConfig", func() {
	Context("when the file does not exist", func() {
		It("returns an error", func() {
			_, err := backup.LoadBackupConfig("/this/is/an/invalid/path")
			Expect(err.Error()).To(Equal("open /this/is/an/invalid/path: no such file or directory"))
		})
	})

	Context("when the file is successfully loaded", func() {
		var config *backup.BackupConfig

		BeforeEach(func() {
			path, err := filepath.Abs(path.Join("assets", "backup.yml"))
			Expect(err).ToNot(HaveOccurred())

			config, err = backup.LoadBackupConfig(path)
			Expect(err).ToNot(HaveOccurred())
		})

		It("has the correct endpoint_url", func() {
			Expect(config.S3Config.EndpointUrl).To(Equal("endpointurl.com"))
		})

		It("has the correct bucket_name", func() {
			Expect(config.S3Config.BucketName).To(Equal("some-bucket-name"))
		})

		It("has the correct access_key_id", func() {
			Expect(config.S3Config.AccessKeyId).To(Equal("some-access-key-id"))
		})

		It("has the correct secret_access_key", func() {
			Expect(config.S3Config.SecretAccessKey).To(Equal("secret-access-key"))
		})

		It("has the correct path", func() {
			Expect(config.S3Config.Path).To(Equal("some-s3-path"))
		})

		It("has the correct snapshot_timeout_seconds", func() {
			Expect(config.SnapshotTimeoutSeconds).To(Equal(10))
		})

		It("has the correct redis_config_root", func() {
			Expect(config.RedisConfigRoot).To(Equal("/the/path/to/redis/config"))
		})

		It("has the correct redis_config_filename", func() {
			Expect(config.RedisConfigFilename).To(Equal("redis-config-filename"))
		})

		It("has the correct broker_address", func() {
			Expect(config.BrokerAddress).To(Equal("localhost:1234"))
		})

		It("has the correct broker credentials", func() {
			Expect(config.BrokerCredentials.Username).To(Equal("admin"))
			Expect(config.BrokerCredentials.Password).To(Equal("secret"))
		})

		It("has the correct node_ip", func() {
			Expect(config.NodeIP).To(Equal("1.2.3.4"))
		})

		It("has the correct plan_name", func() {
			Expect(config.PlanName).To(Equal("plan-name"))
		})

		It("has the correct log file path", func() {
			Expect(config.LogFilepath).To(Equal("/log/file/path"))
		})

		It("has the correct aws cli path", func() {
			Expect(config.AwsCLIPath).To(Equal("/path/to/aws-cli"))
		})
	})
})
