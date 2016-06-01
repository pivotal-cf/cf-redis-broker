package redis_test

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/pborman/uuid"

	"github.com/pivotal-cf/cf-redis-broker/brokerconfig"
	"github.com/pivotal-cf/cf-redis-broker/redis"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Local Repository", func() {
	var instanceID string
	var repo *redis.LocalRepository
	var tmpInstanceDataDir string = "/tmp/repotests/data"
	var tmpInstanceLogDir string = "/tmp/repotests/log"
	var defaultConfigFilePath string = "/tmp/default_config_path"
	var defaultConfigFileContents string = "daemonize yes"

	BeforeEach(func() {
		instanceID = uuid.NewRandom().String()

		// set up default conf file
		file, createFileErr := os.Create(defaultConfigFilePath)
		Ω(createFileErr).ToNot(HaveOccurred())

		_, fileWriteErr := file.WriteString(defaultConfigFileContents)
		Ω(fileWriteErr).ToNot(HaveOccurred())

		repo = &redis.LocalRepository{
			RedisConf: brokerconfig.ServiceConfiguration{
				Host:                  "127.0.0.1",
				DefaultConfigPath:     "/tmp/default_config_path",
				InstanceDataDirectory: tmpInstanceDataDir,
				InstanceLogDirectory:  tmpInstanceLogDir,
			},
		}

		err := os.MkdirAll(tmpInstanceDataDir, 0755)
		Ω(err).ToNot(HaveOccurred())

		err = os.MkdirAll(tmpInstanceLogDir, 0755)
		Ω(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		err := os.RemoveAll(tmpInstanceDataDir)
		Ω(err).ToNot(HaveOccurred())

		err = os.RemoveAll(tmpInstanceLogDir)
		Ω(err).ToNot(HaveOccurred())
	})

	Describe("InstancePid", func() {

		Context("when a pid file exists", func() {
			instanceID := uuid.NewRandom().String()

			instance := redis.Instance{
				ID: instanceID,
			}

			BeforeEach(func() {
				pid := "1234"
				basepath := tmpInstanceDataDir
				instanceDir := path.Join(basepath, instanceID)
				mkdirErr := os.MkdirAll(instanceDir, 0755)
				Ω(mkdirErr).ToNot(HaveOccurred())
				pidFilePath := instanceDir + "/redis-server.pid"
				ioutil.WriteFile(pidFilePath, []byte(pid), 0644)
			})

			It("returns its value", func() {
				pidFromFile, err := repo.InstancePid(instance.ID)
				Ω(err).ToNot(HaveOccurred())
				Ω(pidFromFile).To(Equal(1234))
			})
		})

		Context("when a pid file does not exist", func() {
			instanceID := uuid.NewRandom().String()

			instance := redis.Instance{
				ID: instanceID,
			}

			It("returns an error", func() {
				_, err := repo.InstancePid(instance.ID)
				Ω(err).To(HaveOccurred())
			})
		})
	})

	Describe("reading and writing instances", func() {
		Context("when the repository does not exist", func() {
			It("writes and then reads an instance", func() {
				originalInstance := newTestInstance(instanceID, repo)

				instanceFromDisk, loadInstanceErr := repo.FindByID(instanceID)
				Ω(loadInstanceErr).ToNot(HaveOccurred())

				Ω(instanceFromDisk.ID).To(Equal(originalInstance.ID))
				Ω(instanceFromDisk.Host).To(Equal(originalInstance.Host))
				Ω(instanceFromDisk.Port).To(Equal(originalInstance.Port))
				Ω(instanceFromDisk.Password).To(Equal(originalInstance.Password))
			})

			It("creates the instance data directory", func() {
				newTestInstance(instanceID, repo)

				dataDir := path.Join(tmpInstanceDataDir, instanceID, "db")

				_, err := os.Stat(dataDir)
				Ω(err).NotTo(HaveOccurred())
			})

			It("writes the default config file", func() {
				newTestInstance(instanceID, repo)

				configFilePath := path.Join(tmpInstanceDataDir, instanceID, "redis.conf")

				_, statFileErr := os.Stat(configFilePath)
				Ω(statFileErr).NotTo(HaveOccurred())
			})

			It("creates the instance log directory", func() {
				newTestInstance(instanceID, repo)

				logDir := path.Join(tmpInstanceLogDir, instanceID)

				_, err := os.Stat(logDir)
				Ω(err).NotTo(HaveOccurred())
			})
		})

		Context("when the repository already exists", func() {
			var instance *redis.Instance

			BeforeEach(func() {
				instance = newTestInstance(instanceID, repo)
			})

			It("overwrites the config file", func() {
				originalConfigContents := []byte("my custom config")
				err := ioutil.WriteFile(repo.InstanceConfigPath(instance.ID), originalConfigContents, 0755)
				Ω(err).NotTo(HaveOccurred())

				writeInstance(instance, repo)

				configContents, err := ioutil.ReadFile(repo.InstanceConfigPath(instance.ID))
				Ω(err).NotTo(HaveOccurred())
				Ω(configContents).ShouldNot(Equal(originalConfigContents))
			})

			It("does not clear the data directory", func() {
				dataFilePath := filepath.Join(repo.InstanceDataDir(instance.ID), "appendonly.aof")

				originalDataFileContents := []byte("DATA FILE")
				err := ioutil.WriteFile(dataFilePath, originalDataFileContents, 0755)
				Ω(err).NotTo(HaveOccurred())

				writeInstance(instance, repo)

				dataFileContents, err := ioutil.ReadFile(dataFilePath)
				Ω(err).NotTo(HaveOccurred())
				Ω(dataFileContents).Should(Equal(originalDataFileContents))
			})

			It("does not clear the log directory", func() {
				logFilePath := filepath.Join(repo.InstanceLogDir(instance.ID), "redis-server.log")

				originalLogFileContents := []byte("LOG FILE")
				err := ioutil.WriteFile(logFilePath, originalLogFileContents, 0755)
				Ω(err).NotTo(HaveOccurred())

				writeInstance(instance, repo)

				logFileContents, err := ioutil.ReadFile(logFilePath)
				Ω(err).NotTo(HaveOccurred())
				Ω(logFileContents).Should(Equal(originalLogFileContents))
			})

			Context("when there is no log directory", func() {
				BeforeEach(func() {
					err := os.RemoveAll(repo.InstanceLogDir(instance.ID))
					Ω(err).NotTo(HaveOccurred())
				})

				It("recreates the log directory", func() {
					err := repo.EnsureDirectoriesExist(instance)
					Ω(err).NotTo(HaveOccurred())

					Ω(fileExists(repo.InstanceLogDir(instance.ID))).Should(BeTrue())
				})
			})
		})
	})

	Describe("FindByID", func() {
		Context("when instance does not exist", func() {
			It("returns an error", func() {
				_, err := repo.FindByID(instanceID)
				Ω(err).To(BeAssignableToTypeOf(&os.PathError{}))
			})
		})
	})

	Describe("InstanceExists", func() {
		Context("when instance does not exist", func() {
			It("returns false", func() {
				result, err := repo.InstanceExists(instanceID)
				Ω(result).Should(BeFalse())
				Ω(err).ShouldNot(HaveOccurred())
			})
		})

		Context("when instance exists", func() {
			BeforeEach(func() {
				newTestInstance(instanceID, repo)
			})

			It("returns true", func() {
				result, err := repo.InstanceExists(instanceID)
				Ω(result).Should(BeTrue())
				Ω(err).ShouldNot(HaveOccurred())
			})
		})
	})

	Describe("Delete", func() {
		Context("When the instance exists", func() {
			BeforeEach(func() {
				newTestInstance(instanceID, repo)
			})

			It("deletes the instance data directory", func() {
				repo.Delete(instanceID)
				_, err := os.Stat(path.Join(tmpInstanceDataDir, instanceID))
				Ω(err).To(HaveOccurred())
			})

			It("deletes the instance log directory", func() {
				repo.Delete(instanceID)
				_, err := os.Stat(path.Join(tmpInstanceLogDir, instanceID))
				Ω(err).To(HaveOccurred())
			})

			It("returns no error", func() {
				err := repo.Delete(instanceID)
				Ω(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("InstanceCount", func() {
		Context("when there are no instances", func() {
			It("returns 0", func() {
				instanceCount, err := repo.InstanceCount()
				Ω(err).ToNot(HaveOccurred())
				Ω(instanceCount).To(Equal(0))
			})
		})

		Context("when there are some instances", func() {
			It("returns the correct count", func() {
				newTestInstance(instanceID, repo)

				instanceCount, err := repo.InstanceCount()
				Ω(err).ToNot(HaveOccurred())
				Ω(instanceCount).To(Equal(1))
			})
		})

		Context("when getting the data directories fails", func() {
			It("returns an error", func() {
				os.RemoveAll(tmpInstanceDataDir)

				_, err := repo.InstanceCount()
				Ω(err).To(HaveOccurred())
			})
		})
	})

	Describe("AllInstances", func() {
		Context("when there are no instances", func() {
			It("returns an empty instance slice", func() {
				instances, err := repo.AllInstances()
				Ω(err).ToNot(HaveOccurred())
				Ω(instances).To(BeEmpty())
			})
		})

		Context("when there are some instances", func() {
			It("contains created instances", func() {
				instance := newTestInstance(instanceID, repo)

				instances, err := repo.AllInstances()
				Ω(err).ToNot(HaveOccurred())
				Ω(instances).To(ContainElement(instance))
			})
		})

		It("does not contain deleted instances", func() {
			instance := newTestInstance(instanceID, repo)
			repo.Delete(instanceID)

			instances, err := repo.AllInstances()
			Ω(err).ToNot(HaveOccurred())
			Ω(instances).ToNot(ContainElement(instance))
		})

		Context("when getting the data directories fails", func() {
			It("returns an error", func() {
				os.RemoveAll(tmpInstanceDataDir)

				_, err := repo.AllInstances()
				Ω(err).To(HaveOccurred())
			})
		})
	})
})

func newTestInstance(instanceID string, repo *redis.LocalRepository) *redis.Instance {
	instance := &redis.Instance{
		ID:   instanceID,
		Host: "127.0.0.1",
		Port: 8080,
	}
	writeInstance(instance, repo)
	return instance
}

func writeInstance(instance *redis.Instance, repo *redis.LocalRepository) {
	err := repo.EnsureDirectoriesExist(instance)
	Ω(err).NotTo(HaveOccurred())
	err = repo.WriteConfigFile(instance)
	Ω(err).NotTo(HaveOccurred())
	file, err := os.Create(filepath.Join(repo.InstanceBaseDir(instance.ID), "monitor"))
	Ω(err).NotTo(HaveOccurred())
	file.Close()
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}
