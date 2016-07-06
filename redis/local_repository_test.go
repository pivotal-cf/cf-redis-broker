package redis_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"

	"github.com/pborman/uuid"
	"github.com/pivotal-golang/lager/lagertest"

	"github.com/pivotal-cf/cf-redis-broker/brokerconfig"
	"github.com/pivotal-cf/cf-redis-broker/redis"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Local Repository", func() {
	var instanceID string
	var repo *redis.LocalRepository
	var logger *lagertest.TestLogger
	var tmpInstanceDataDir string = "/tmp/repotests/data"
	var tmpInstanceLogDir string = "/tmp/repotests/log"
	var tmpPidFileDir string = "/tmp/pidfiles"
	var defaultConfigFilePath string = "/tmp/default_config_path"
	var defaultConfigFileContents string = "daemonize yes"

	BeforeEach(func() {
		instanceID = uuid.NewRandom().String()
		logger = lagertest.NewTestLogger("local-repo")

		// set up default conf file
		file, createFileErr := os.Create(defaultConfigFilePath)
		Ω(createFileErr).ToNot(HaveOccurred())

		_, fileWriteErr := file.WriteString(defaultConfigFileContents)
		Ω(fileWriteErr).ToNot(HaveOccurred())

		redisConf := brokerconfig.ServiceConfiguration{
			Host:                  "127.0.0.1",
			DefaultConfigPath:     "/tmp/default_config_path",
			InstanceDataDirectory: tmpInstanceDataDir,
			PidfileDirectory:      tmpPidFileDir,
			InstanceLogDirectory:  tmpInstanceLogDir,
		}

		repo = redis.NewLocalRepository(redisConf, logger)

		err := os.MkdirAll(tmpInstanceDataDir, 0755)
		Ω(err).ToNot(HaveOccurred())

		err = os.MkdirAll(tmpPidFileDir, 0755)
		Ω(err).ToNot(HaveOccurred())

		err = os.MkdirAll(tmpInstanceLogDir, 0755)
		Ω(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		err := os.RemoveAll(tmpInstanceDataDir)
		Ω(err).ToNot(HaveOccurred())

		err = os.RemoveAll(tmpPidFileDir)
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
				pidFilePath := tmpPidFileDir + "/" + instanceID + ".pid"
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

					Expect(repo.InstanceLogDir(instance.ID)).To(BeAnExistingFile())
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

			It("deletes the instance pid file", func() {
				repo.Delete(instanceID)
				_, err := os.Stat(path.Join(tmpPidFileDir, instanceID+".pid"))
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
				instanceCount, errs := repo.InstanceCount()
				Ω(errs).To(BeEmpty())
				Ω(instanceCount).To(Equal(0))
			})
		})

		Context("when there are some instances", func() {
			It("returns the correct count", func() {
				newTestInstance(instanceID, repo)

				instanceCount, errs := repo.InstanceCount()
				Ω(errs).To(BeEmpty())
				Ω(instanceCount).To(Equal(1))
			})
		})

		Context("when getting the data directories fails", func() {
			It("returns an error", func() {
				os.RemoveAll(tmpInstanceDataDir)

				_, errs := repo.InstanceCount()
				Expect(len(errs)).To(Equal(1))
				Ω(errs[0]).To(HaveOccurred())
			})
		})
	})

	Describe("AllInstancesVerbose", func() {
		It("logs that it is starting to look for shared instances, and in which directory", func() {
			repo.AllInstancesVerbose()
			Expect(logger).To(gbytes.Say(fmt.Sprintf("Starting shared instance lookup in data directory: %s", tmpInstanceDataDir)))
		})

		Context("when there are no instances", func() {
			var instances []*redis.Instance

			JustBeforeEach(func() {
				var allInstancesErrors []error
				instances, allInstancesErrors = repo.AllInstancesVerbose()
				Expect(allInstancesErrors).To(BeEmpty())
			})

			It("returns an empty instance slice", func() {
				Expect(instances).To(BeEmpty())
			})

			It("logs the instance count", func() {
				Expect(logger).To(gbytes.Say("0 shared Redis instances found"))
			})
		})

		Context("when there is one instance", func() {
			var instance *redis.Instance
			var instances []*redis.Instance
			var errs []error

			BeforeEach(func() {
				instance = newTestInstance(instanceID, repo)
			})

			Context("listing InstanceExists", func() {
				BeforeEach(func() {
					instances, errs = repo.AllInstancesVerbose()
				})

				It("contains created instances", func() {
					Ω(errs).To(BeEmpty())
					Ω(instances).To(ContainElement(instance))
				})

				It("logs the instance count", func() {
					Expect(logger).To(gbytes.Say("1 shared Redis instance found"))
				})

				It("logs the ID of the instance", func() {
					Expect(logger).To(gbytes.Say(fmt.Sprintf("Found shared instance: %s", instance.ID)))
				})
			})

			Context("when getting one repo ID fails", func() {
				BeforeEach(func() {
					os.Remove(repo.InstanceConfigPath(instance.ID))
					_, errs = repo.AllInstances()
				})

				It("returns one error", func() {
					Expect(len(errs)).To(Equal(1))
					Expect(errs[0]).To(HaveOccurred())
				})

				It("logs the error", func() {
					Expect(logger).To(gbytes.Say(errs[0].Error()))
					Expect(logger).To(gbytes.Say(fmt.Sprintf("Error getting instance details for instance ID: %s", instanceID)))
				})
			})
		})

		Context("when there are several instances", func() {
			var instanceIDs []string
			var instances []*redis.Instance

			BeforeEach(func() {
				for i := 0; i < 3; i++ {
					instanceIDs = append(instanceIDs, uuid.NewRandom().String())
				}
				sort.Strings(instanceIDs)
				for _, instanceID := range instanceIDs {
					instances = append(instances, newTestInstance(instanceID, repo))
				}
			})

			AfterEach(func() {
				instanceIDs = []string{}
				instances = []*redis.Instance{}
			})

			It("logs the instance count", func() {
				_, errs := repo.AllInstancesVerbose()
				Ω(errs).To(BeEmpty())
				Expect(logger).To(gbytes.Say("3 shared Redis instances found"))
			})

			Context("when getting one repo ID fails", func() {
				var errs []error

				BeforeEach(func() {
					os.Remove(repo.InstanceConfigPath(instanceIDs[0]))
					instances, errs = repo.AllInstances()
				})

				It("returns one error", func() {
					Expect(len(errs)).To(Equal(1))
					Expect(errs[0]).To(HaveOccurred())
				})

				It("returns the other two instances", func() {
					Expect(len(instances)).To(Equal(2))
				})
			})
		})

		It("does not contain deleted instances", func() {
			instance := newTestInstance(instanceID, repo)
			repo.Delete(instanceID)

			instances, errs := repo.AllInstances()
			Ω(errs).To(BeEmpty())
			Ω(instances).ToNot(ContainElement(instance))
		})

		Context("when getting the data directories fails", func() {
			It("returns an error", func() {
				os.RemoveAll(tmpInstanceDataDir)

				_, errs := repo.AllInstances()
				Expect(len(errs)).To(Equal(1))
				Ω(errs[0]).To(HaveOccurred())
			})

			It("logs the error", func() {
				os.RemoveAll(tmpInstanceDataDir)

				_, errs := repo.AllInstances()

				Expect(logger).To(gbytes.Say(errs[0].Error()))
				Expect(logger).To(gbytes.Say("Error finding shared instances"))
			})
		})
	})

	Describe("AllInstances", func() {
		It("doesn't log that it is starting to look for shared instances, and in which directory", func() {
			repo.AllInstances()
			Expect(logger).NotTo(gbytes.Say(fmt.Sprintf("Starting shared instance lookup in data directory: %s", tmpInstanceDataDir)))
		})

		Context("when there are no instances", func() {
			It("returns an empty instance slice", func() {
				instances, errs := repo.AllInstances()
				Ω(errs).To(BeEmpty())
				Ω(instances).To(BeEmpty())
			})

			It("doesn't log the instance count", func() {
				Expect(logger).NotTo(gbytes.Say("0 shared Redis instances found"))
			})
		})

		Context("when there is one instance", func() {
			var instance *redis.Instance
			var instances []*redis.Instance
			var errs []error

			BeforeEach(func() {
				instance = newTestInstance(instanceID, repo)
			})

			Context("listing InstanceExists", func() {
				BeforeEach(func() {
					instances, errs = repo.AllInstances()
				})

				It("contains created instances", func() {
					Ω(errs).To(BeEmpty())
					Ω(instances).To(ContainElement(instance))
				})

				It("doesn't log the ID of the instance", func() {
					Expect(logger).NotTo(gbytes.Say(fmt.Sprintf("Found shared instance: %s", instance.ID)))
				})
			})

			Context("when getting one repo ID fails", func() {
				BeforeEach(func() {
					os.Remove(repo.InstanceConfigPath(instance.ID))
					_, errs = repo.AllInstances()
				})

				It("returns one error", func() {
					Expect(len(errs)).To(Equal(1))
					Expect(errs[0]).To(HaveOccurred())
				})

				It("logs the error", func() {
					Expect(logger).To(gbytes.Say(errs[0].Error()))
					Expect(logger).To(gbytes.Say(fmt.Sprintf("Error getting instance details for instance ID: %s", instanceID)))
				})
			})
		})

		Context("when there are several instances", func() {
			var instanceIDs []string
			var instances []*redis.Instance

			BeforeEach(func() {
				for i := 0; i < 3; i++ {
					instanceIDs = append(instanceIDs, uuid.NewRandom().String())
				}
				sort.Strings(instanceIDs)
				for _, instanceID := range instanceIDs {
					instances = append(instances, newTestInstance(instanceID, repo))
				}
			})

			AfterEach(func() {
				instanceIDs = []string{}
				instances = []*redis.Instance{}
			})

			It("doesn't log the instance count", func() {
				_, errs := repo.AllInstances()
				Ω(errs).To(BeEmpty())
				Expect(logger).NotTo(gbytes.Say("3 shared Redis instances found"))
			})

			Context("when getting one repo ID fails", func() {
				var errs []error

				BeforeEach(func() {
					os.Remove(repo.InstanceConfigPath(instanceIDs[0]))
					instances, errs = repo.AllInstances()
				})

				It("returns one error", func() {
					Expect(len(errs)).To(Equal(1))
					Expect(errs[0]).To(HaveOccurred())
				})

				It("returns the other two instances", func() {
					Expect(len(instances)).To(Equal(2))
				})
			})
		})

		It("does not contain deleted instances", func() {
			instance := newTestInstance(instanceID, repo)
			repo.Delete(instanceID)

			instances, errs := repo.AllInstances()
			Ω(errs).To(BeEmpty())
			Ω(instances).ToNot(ContainElement(instance))
		})

		Context("when getting the data directories fails", func() {
			It("returns an error", func() {
				os.RemoveAll(tmpInstanceDataDir)

				_, errs := repo.AllInstances()
				Expect(len(errs)).To(Equal(1))
				Ω(errs[0]).To(HaveOccurred())
			})

			It("logs the error", func() {
				os.RemoveAll(tmpInstanceDataDir)

				_, errs := repo.AllInstances()

				Expect(logger).To(gbytes.Say(errs[0].Error()))
				Expect(logger).To(gbytes.Say("Error finding shared instances"))
			})
		})
	})
})

var _ = Describe("Setup", func() {
	var repo *redis.LocalRepository
	var instanceID string
	var logger *lagertest.TestLogger
	var tmpConfigFilePath string = "/tmp/default_config_path"
	var tmpDataDir string = "/tmp/repotests/data"
	var tmpPidfileDir string = "/tmp/repotests/pids"
	var tmpLogDir string = "/tmp/repotests/log"
	var instance redis.Instance

	BeforeEach(func() {
		err := os.MkdirAll(tmpDataDir, 0755)
		Expect(err).ToNot(HaveOccurred())
		err = os.MkdirAll(tmpPidfileDir, 0755)
		Expect(err).ToNot(HaveOccurred())
		err = os.MkdirAll(tmpLogDir, 0755)
		Expect(err).ToNot(HaveOccurred())
		_, createFileErr := os.Create(tmpConfigFilePath)
		Expect(createFileErr).ToNot(HaveOccurred())

		instanceID = uuid.NewRandom().String()
		logger = lagertest.NewTestLogger("local-repo-setup")

		instance = redis.Instance{
			ID: instanceID,
		}

		redisConf := brokerconfig.ServiceConfiguration{
			Host:                  "127.0.0.1",
			DefaultConfigPath:     tmpConfigFilePath,
			InstanceDataDirectory: tmpDataDir,
			PidfileDirectory:      tmpPidfileDir,
			InstanceLogDirectory:  tmpLogDir,
		}

		repo = redis.NewLocalRepository(redisConf, logger)
	})

	AfterEach(func() {
		err := os.RemoveAll(tmpDataDir)
		Expect(err).ToNot(HaveOccurred())

		err = os.RemoveAll(tmpPidfileDir)
		Expect(err).ToNot(HaveOccurred())

		err = os.RemoveAll(tmpLogDir)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("When setup is successful", func() {
		It("creates the appropriate directories", func() {
			tmpInstanceDataDir := path.Join(tmpDataDir, instanceID)
			tmpInstanceLogDir := path.Join(tmpLogDir, instanceID)
			Expect(tmpInstanceDataDir).NotTo(BeADirectory())
			Expect(tmpInstanceLogDir).NotTo(BeADirectory())

			err := repo.Setup(&instance)
			Expect(err).NotTo(HaveOccurred())

			Expect(tmpInstanceDataDir).To(BeADirectory())
			Expect(tmpInstanceLogDir).To(BeADirectory())
		})

		It("creates a lock file", func() {
			err := repo.Setup(&instance)
			Expect(err).NotTo(HaveOccurred())

			lockFilePath := path.Join(tmpDataDir, instanceID, "lock")
			Expect(lockFilePath).To(BeAnExistingFile())
		})

		It("writes the config file", func() {
			err := repo.Setup(&instance)
			Expect(err).NotTo(HaveOccurred())

			configFilePath := path.Join(tmpDataDir, instanceID, "redis.conf")

			configFileContent, err := ioutil.ReadFile(configFilePath)
			Expect(err).NotTo(HaveOccurred())

			redisServerName := "redis-server-" + instanceID
			Expect(configFileContent).To(ContainSubstring(redisServerName))
		})
	})

	Context("When setup is not successful", func() {
		Context("the instance dir does not have write permissions", func() {
			BeforeEach(func() {
				err := os.Chmod(tmpDataDir, 0400)
				Expect(err).NotTo(HaveOccurred())
				err = os.Chmod(tmpLogDir, 0400)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns an error", func() {
				err := repo.Setup(&instance)
				Expect(err).To(HaveOccurred())
			})

			It("logs the error", func() {
				_ = repo.Setup(&instance)

				Expect(logger).To(gbytes.Say("local-repo-setup.ensure-dirs-exist"))
				Expect(logger).To(gbytes.Say("permission denied"))
			})

			AfterEach(func() {
				err := os.Chmod(tmpDataDir, 0755)
				Expect(err).NotTo(HaveOccurred())
				err = os.Chmod(tmpLogDir, 0755)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("the config file being written is invalid", func() {
			var invalidConfigFilePath string

			BeforeEach(func() {
				invalidConfigFilePath = "/tmp/invalid_config_path"
				invalidConfigFileContents := "notavalidconfig"

				file, createFileErr := os.Create(invalidConfigFilePath)
				Ω(createFileErr).ToNot(HaveOccurred())

				_, fileWriteErr := file.WriteString(invalidConfigFileContents)
				Ω(fileWriteErr).ToNot(HaveOccurred())

				repo.RedisConf.DefaultConfigPath = invalidConfigFilePath
			})

			It("returns an error", func() {
				err := repo.Setup(&instance)
				Expect(err).To(HaveOccurred())
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
	pid := []byte("1234")
	err = ioutil.WriteFile(repo.InstancePidFilePath(instance.ID), pid, 0644)
	Ω(err).NotTo(HaveOccurred())
	file.Close()
}
