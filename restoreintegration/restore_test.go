package restoreintegration_test

import (
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf/cf-redis-broker/restoreconfig"
)

const (
	sharedPlan    string = "shared"
	dedicatedPlan string = "dedicated"
)

var _ = Describe("restore", func() {
	var restoreCommand *exec.Cmd

	var instanceID string
	var sourceRdbPath string
	var redisSession *gexec.Session
	var monitLogFile string

	var config restoreconfig.Config

	BeforeEach(func() {
		instanceID = "test_instance"
		sourceRdbPath = filepath.Join("assets", "dump.rdb")
		err := copyFile(filepath.Join("..", "brokerintegration", "assets", "redis.conf"), "/tmp/redis.conf")
		Ω(err).ShouldNot(HaveOccurred())
		err = copyFile(filepath.Join("assets", "monit"), "/tmp/monit")
		Ω(err).ShouldNot(HaveOccurred())
		err = os.Chmod("/tmp/monit", 0755)
		Ω(err).ShouldNot(HaveOccurred())
		monitLogFile = setupMonitLogFile()
	})

	AfterEach(func() {
		pid, err := config.InstancePid(instanceID)
		if err == nil {
			syscall.Kill(pid, syscall.SIGKILL)
		}

		Eventually(redisSession, "20s").Should(gexec.Exit(0))
	})

	Describe("common to plans", func() {
		BeforeEach(func() {
			config = loadRestoreConfig(sharedPlan)
			redisSession = startRedisSession(config, instanceID, sharedPlan)
			restoreCommand = buildRestoreCommand(sourceRdbPath, monitLogFile, instanceID, sharedPlan)
		})

		It("exits with a non zero status if no arguments are provided", func() {
			restoreCommand.Args = []string{}
			session, err := gexec.Start(restoreCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session, "20s").Should(gexec.Exit(1))
			Eventually(session.Err).Should(gbytes.Say("usage: restore <instance_id> <rdb_path>"))
		})

		It("exits with a non zero status if the RDB file does not exist", func() {
			restoreCommand.Args = []string{restoreCommand.Args[0], instanceID, "bar"}
			session, err := gexec.Start(restoreCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session.Err).Should(gbytes.Say("RDB file not found"))
			Eventually(session, "20s").Should(gexec.Exit(1))
		})

		It("exits with a non zero status if the config cannot be loaded", func() {
			restoreCommand.Env = []string{"RESTORE_CONFIG_PATH=foo"}
			session, err := gexec.Start(restoreCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session.Err).Should(gbytes.Say("Could not load config"))
			Eventually(session, "20s").Should(gexec.Exit(1))
		})

		It("stops redis", func() {
			session, err := gexec.Start(restoreCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(redisSession, "20s").Should(gexec.Exit(0))
			Eventually(session, "20s").Should(gexec.Exit(0))
		})

		It("exits successfully if the instance and the RDB file exist", func() {
			session, err := gexec.Start(restoreCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session, "20s").Should(gexec.Exit(0))
		})

		It("does not leave redis running", func() {
			session, err := gexec.Start(restoreCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session, "20s").Should(gexec.Exit(0))

			pkillCommand := exec.Command("pkill", "redis-server")
			pkillSession, err := gexec.Start(pkillCommand, GinkgoWriter, GinkgoWriter)
			// pkill returns 1 if there is nothing for it to kill
			Eventually(pkillSession).Should(gexec.Exit(1))
		})
	})

	Describe("shared plan", func() {
		BeforeEach(func() {
			config = loadRestoreConfig(sharedPlan)
			redisSession = startRedisSession(config, instanceID, sharedPlan)
			restoreCommand = buildRestoreCommand(sourceRdbPath, monitLogFile, instanceID, sharedPlan)
		})

		It("exits with a non zero status if the instance directory does not exist", func() {
			restoreCommand.Args = []string{restoreCommand.Args[0], "foo", sourceRdbPath}
			session, err := gexec.Start(restoreCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session.Err).Should(gbytes.Say("Instance not found"))
			Eventually(session, "20s").Should(gexec.Exit(1))
		})

		It("stops and then starts the process-watcher", func() {
			session, err := gexec.Start(restoreCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session, "20s").Should(gexec.Exit(0))

			monitLogBytes, err := ioutil.ReadFile(monitLogFile)
			Expect(err).ToNot(HaveOccurred())

			Expect(string(monitLogBytes)).To(ContainSubstring("stopping process-watcher"))
			Expect(string(monitLogBytes)).To(ContainSubstring("starting process-watcher"))
		})

		It("creates a new RDB file in the instance directory", func() {
			newRdbPath := filepath.Join(config.InstanceDataDir(instanceID), "dump.rdb")

			_, err := os.Stat(newRdbPath)
			Expect(os.IsNotExist(err)).To(BeTrue())

			session, err := gexec.Start(restoreCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session, "20s").Should(gexec.Exit(0))

			copiedFileContents, err := ioutil.ReadFile(newRdbPath)
			Expect(err).NotTo(HaveOccurred())
			sourceFileContents, err := ioutil.ReadFile(sourceRdbPath)
			Expect(err).NotTo(HaveOccurred())

			Expect(copiedFileContents).To(Equal(sourceFileContents))
		})

		It("creates a new AOF file in the instance directory", func() {
			aofPath := filepath.Join(config.InstanceDataDir(instanceID), "appendonly.aof")

			_, err := os.Stat(aofPath)
			Expect(os.IsNotExist(err)).To(BeTrue())

			session, err := gexec.Start(restoreCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session, "20s").Should(gexec.Exit(0))

			fileContents, err := ioutil.ReadFile(aofPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(fileContents)).To(ContainSubstring("TEST_KEY"))
		})

	})

	Describe("dedicated plan", func() {
		BeforeEach(func() {
			config = loadRestoreConfig(dedicatedPlan)
			redisSession = startRedisSession(config, instanceID, dedicatedPlan)
			restoreCommand = buildRestoreCommand(sourceRdbPath, monitLogFile, instanceID, dedicatedPlan)
		})

		It("doesnt stop and then start the process-watcher", func() {
			session, err := gexec.Start(restoreCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session, "20s").Should(gexec.Exit(0))

			monitLogBytes, err := ioutil.ReadFile(monitLogFile)
			Expect(err).ToNot(HaveOccurred())

			Expect(string(monitLogBytes)).ToNot(ContainSubstring("stopping process-watcher"))
			Expect(string(monitLogBytes)).ToNot(ContainSubstring("starting process-watcher"))
		})

		It("it tells monit to unmonitor redis", func() {
			session, err := gexec.Start(restoreCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session, "20s").Should(gexec.Exit(0))

			monitLogBytes, err := ioutil.ReadFile(monitLogFile)
			Expect(err).ToNot(HaveOccurred())

			Expect(string(monitLogBytes)).To(ContainSubstring("unmonitoring redis"))
		})

		It("it tells monit to start redis", func() {
			session, err := gexec.Start(restoreCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session, "20s").Should(gexec.Exit(0))

			monitLogBytes, err := ioutil.ReadFile(monitLogFile)
			Expect(err).ToNot(HaveOccurred())

			Expect(string(monitLogBytes)).To(ContainSubstring("starting redis"))
		})

		It("creates a new RDB file in the instance directory", func() {
			newRdbPath := filepath.Join(config.InstanceDataDir(instanceID), "dump.rdb")

			_, err := os.Stat(newRdbPath)
			Expect(os.IsNotExist(err)).To(BeTrue())

			session, err := gexec.Start(restoreCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session, "20s").Should(gexec.Exit(0))

			copiedFileContents, err := ioutil.ReadFile(newRdbPath)
			Expect(err).NotTo(HaveOccurred())
			sourceFileContents, err := ioutil.ReadFile(sourceRdbPath)
			Expect(err).NotTo(HaveOccurred())

			Expect(copiedFileContents).To(Equal(sourceFileContents))
		})

		It("creates a new AOF file in the instance directory", func() {
			aofPath := filepath.Join(config.InstanceDataDir(instanceID), "appendonly.aof")

			_, err := os.Stat(aofPath)
			Expect(os.IsNotExist(err)).To(BeTrue())

			session, err := gexec.Start(restoreCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session, "20s").Should(gexec.Exit(0))

			fileContents, err := ioutil.ReadFile(aofPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(fileContents)).To(ContainSubstring("TEST_KEY"))
		})
	})
})

func startRedisSession(config restoreconfig.Config, instanceID, planName string) (redisSession *gexec.Session) {
	var testInstanceDir string
	testDataDir := config.InstanceDataDir(instanceID)
	if planName == dedicatedPlan {
		testInstanceDir = testDataDir
		os.RemoveAll(testDataDir)
	} else {
		testInstanceDir = filepath.Join(config.RedisDataDirectory, instanceID)
		os.RemoveAll(testInstanceDir)
	}
	os.MkdirAll(testDataDir, 0777)

	err := ioutil.WriteFile(
		filepath.Join(testInstanceDir, "redis.conf"),
		[]byte("port 6379"),
		os.ModePerm,
	)
	Expect(err).ToNot(HaveOccurred())

	pidfilePath := config.InstancePidFilePath(instanceID)

	err = os.MkdirAll(config.PidfileDirectory, 0777)
	Expect(err).NotTo(HaveOccurred())

	redisCmd := exec.Command("redis-server",
		"--dir", testInstanceDir,
		"--save", "900", "1",
		"--pidfile", pidfilePath,
		"--daemonize", "yes",
	)

	redisSession, err = gexec.Start(redisCmd, GinkgoWriter, GinkgoWriter)
	Expect(err).ToNot(HaveOccurred())

	pidFileWritten := make(chan bool)
	go func(c chan<- bool) {
		for {
			if _, err := os.Stat(pidfilePath); !os.IsNotExist(err) {
				c <- true
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
	}(pidFileWritten)

	// wait for redis to write pid file
	select {
	case <-pidFileWritten:
		break
	case <-time.After(30 * time.Second):
		Fail("Test timed out waiting for redis to write PID file.")
	}

	return redisSession
}

func loadRestoreConfig(planName string) restoreconfig.Config {
	configPath := filepath.Join("assets", "restore-"+planName+".yml")

	config, err := restoreconfig.Load(configPath)
	Expect(err).ToNot(HaveOccurred())
	return config
}

func buildRestoreCommand(sourceRdbPath, monitLogFile, instanceID, planName string) *exec.Cmd {
	configPath := filepath.Join("assets", "restore-"+planName+".yml")
	restoreCommand := exec.Command(restoreExecutablePath, instanceID, sourceRdbPath)
	restoreCommand.Env = append(os.Environ(), "RESTORE_CONFIG_PATH="+configPath)
	restoreCommand.Env = append(restoreCommand.Env, "MONIT_LOG_FILE="+monitLogFile)

	fakeChownPath := "assets"
	for i, envVar := range restoreCommand.Env {
		parts := strings.Split(envVar, "=")
		if parts[0] == "PATH" {
			path := fakeChownPath + ":" + parts[1]

			restoreCommand.Env[i] = "PATH=" + path
		}
	}
	return restoreCommand
}

func copyFile(sourcePath, destinationPath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(destinationPath)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

func setupMonitLogFile() string {
	monitLogDir, err := ioutil.TempDir("", "monit-test-logs")
	Expect(err).NotTo(HaveOccurred())

	return filepath.Join(monitLogDir, "monit.log")
}
