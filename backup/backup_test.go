package backup

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/cf-redis-broker/backupconfig"
	"github.com/pivotal-cf/cf-redis-broker/redis/client"
	"github.com/pivotal-cf/cf-redis-broker/redis/client/fakes"
)

var _ = Describe("backup", func() {
	Describe(".createSnapshot", func() {
		var (
			numConnectCalls int
			snapshotErr     error
			redisConfPath   string
			redisClient     *fakes.Client
		)

		BeforeEach(func() {
			numConnectCalls = 0

			redisConf, err := ioutil.TempFile("", "redis")
			Expect(err).ToNot(HaveOccurred())
			redisConfPath = redisConf.Name()

			err = redisConf.Close()
			Expect(err).ToNot(HaveOccurred())

			redisClient = &fakes.Client{}
			redisConnect = func(options ...client.Option) (client.Client, error) {
				numConnectCalls++
				return redisClient, nil
			}
		})

		JustBeforeEach(func() {
			backup := Backup{
				Config: &backupconfig.Config{
					BGSaveTimeoutSeconds: 100,
				},
			}
			snapshotErr = backup.createSnapshot(redisConfPath)
		})

		It("does not return an error", func() {
			Expect(snapshotErr).ToNot(HaveOccurred())
		})

		It("connects to redis", func() {
			Expect(numConnectCalls).To(Equal(1))
		})

		It("takes a snapshot", func() {
			Expect(redisClient.CreateSnapshotCalls).To(Equal(1))
		})

		Context("redis config can not be found", func() {
			BeforeEach(func() {
				redisConfPath = "non-existent"
			})

			It("returns an error", func() {
				Expect(snapshotErr).To(HaveOccurred())
				Expect(snapshotErr.Error()).To(ContainSubstring("no such file"))
			})
		})

		Context("an error occurs during redis connect", func() {
			var expectErr = errors.New("failed to connect")

			BeforeEach(func() {
				redisConnect = func(options ...client.Option) (client.Client, error) {
					return nil, expectErr
				}
			})

			It("returns the error", func() {
				Expect(snapshotErr).To(Equal(expectErr))
			})
		})

		Context("an error occurs during redis snapshotting", func() {
			var expectedErr = errors.New("snapshot error")
			BeforeEach(func() {
				redisClient.ExpectedCreateSnapshotErr = expectedErr
			})

			It("returns the error", func() {
				Expect(snapshotErr).To(Equal(expectedErr))
			})
		})
	})

	Describe(".cleanup", func() {
		var tempRdbFile *os.File

		BeforeEach(func() {
			var err error
			tempRdbFile, err = ioutil.TempFile("", "temp")
			Expect(err).ToNot(HaveOccurred())
		})

		Context("RDB file does not exist", func() {
			It("renames the temp RDB", func() {
				dir, err := ioutil.TempDir("", "")
				Expect(err).ToNot(HaveOccurred())

				rdbPath := filepath.Join(dir, "dump.rdb")

				Expect(fileExists(rdbPath)).To(BeFalse())

				cleanup(tempRdbFile.Name(), rdbPath)

				Expect(fileExists(tempRdbFile.Name())).To(BeFalse())
				Expect(fileExists(rdbPath)).To(BeTrue())
			})
		})

		Context("RDB file exists", func() {
			var rdbPath string

			BeforeEach(func() {
				rdbFile, err := ioutil.TempFile("", "rdb")
				Expect(err).ToNot(HaveOccurred())
				_, err = rdbFile.WriteString("something")
				Expect(err).ToNot(HaveOccurred())
				rdbPath = rdbFile.Name()
				rdbFile.Close()
			})

			It("does not touch the existing RDB", func() {
				cleanup(tempRdbFile.Name(), rdbPath)

				Expect(fileExists(rdbPath)).To(BeTrue())
				fileContents, err := ioutil.ReadFile(rdbPath)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(fileContents)).To(Equal("something"))
			})

			It("deletes the temp RDB", func() {
				cleanup(tempRdbFile.Name(), rdbPath)
				Expect(fileExists(tempRdbFile.Name())).To(BeFalse())
			})
		})
	})
})
