package backup_integration_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/cf-cassandra-broker/test/helpers"
)

var _ = Describe("backups", func() {
	Context("when S3 is not configured", func() {
		It("exits with status code 0", func() {
			configFile := helpers.AssetPath("empty-backup.yml")
			backupExitCode := runBackup(configFile)
			Expect(backupExitCode).Should(Equal(0))
		})
	})

	/*Context("when S3 is configured", func() {
		Context("when its a dedicated instance to back up", func() {
			It("creates a dump.rdb file in the redis data dir", func() {
				backupExitCode := runBackup(filepath.Join("assets", "working-backup.yml"))
				Expect(backupExitCode).Should(Equal(0))
				_, err := os.Stat("/tmp/redis-data-dir/file")
				Expect(err).ToNot(HaveOccurred())
			})

			It("uploads the dump.rdb file to the correct S3 bucket", func() {
			})

			// Context("when broker is not responding", func() {
			// 	It("returns non-zero exit code", func() {
			// 	})
			// })

			// Context("when broker returns an error", func() {
			// 	It("returns non-zero exit code", func() {
			// 	})
			// })

			Context("when the instance backup fails", func() {
				It("returns non-zero exit code", func() {
				})
			})
		})

		Context("when there are shared-vm instances to back up", func() {
			Context("when the backup command completes successfully", func() {
				It("exits with status code 0", func() {
				})

				It("uploads a dump.rdb file to S3 for each Redis instance", func() {
				})

				It("creates a dump.rdb file for each Redis instance", func() {
				})
			})

			Context("when an instance backup fails", func() {
				It("still backs up the other instances", func() {
				})
			})
		})
	})*/
})
