package resetter_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/pivotal-cf/cf-redis-broker/redisconf"
	"github.com/pivotal-cf/cf-redis-broker/resetter"
	monitFakes "github.com/pivotal-cf/redisutils/monit/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type fakeChecker struct {
	addressesWaitedOn []*net.TCPAddr
	checkErr          error
}

func (portChecker *fakeChecker) Check(address *net.TCPAddr, timeout time.Duration) error {
	portChecker.addressesWaitedOn = append(portChecker.addressesWaitedOn, address)
	return portChecker.checkErr
}

type fakeRunner struct {
	commandsRan          []*exec.Cmd
	summaryCallCount     int
	summaryOutputStrings []string
	redisProcessStatus   string
}

func (commandRunner *fakeRunner) Run(command *exec.Cmd) ([]byte, error) {
	commandRunner.commandsRan = append(commandRunner.commandsRan, command)

	if command.Args[1] == "stop" {
		commandRunner.redisProcessStatus = "not monitored"
	} else if command.Args[1] == "start" {
		commandRunner.redisProcessStatus = "running"
	} else if command.Args[1] == "summary" {
		commandRunner.summaryCallCount++

		if len(commandRunner.summaryOutputStrings) == 0 {
			return []byte("Process 'redis' " + commandRunner.redisProcessStatus), nil
		}

		return []byte(commandRunner.summaryOutputStrings[commandRunner.summaryCallCount-1]), nil
	}

	return []byte{}, errors.New("Command not recognised by fake")
}

var _ = Describe("Client", func() {
	var (
		redisClient     *resetter.Resetter
		fakePortChecker *fakeChecker
		commandRunner   *fakeRunner
		aofPath         string
		rdbPath         string
		redisPort       int
		confPath        string
		defaultConfPath string
		conf            redisconf.Conf
		fakeMonit       *monitFakes.FakeMonit

		monitExecutablePath = "/path/to/monit"
		redisPassword       = "somepassword"
	)

	BeforeEach(func() {
		fakeMonit = new(monitFakes.FakeMonit)
		commandRunner = new(fakeRunner)
		commandRunner.redisProcessStatus = "running"
		fakePortChecker = new(fakeChecker)

		tmpdir, err := ioutil.TempDir("", "redisconf-test")
		Ω(err).ToNot(HaveOccurred())
		defaultConfPath = filepath.Join(tmpdir, "redis.conf-default")
		confPath = filepath.Join(tmpdir, "redis.conf")

		err = redisconf.New(
			redisconf.Param{
				Key:   "port",
				Value: fmt.Sprintf("%d", redisPort),
			},
			redisconf.Param{
				Key:   "requirepass",
				Value: "default",
			},
		).Save(defaultConfPath)
		Ω(err).ToNot(HaveOccurred())

		conf = redisconf.New(
			redisconf.Param{
				Key:   "port",
				Value: fmt.Sprintf("%d", redisPort),
			},
			redisconf.Param{
				Key:   "appendonly",
				Value: "no",
			},
			redisconf.Param{
				Key:   "requirepass",
				Value: redisPassword,
			},
			redisconf.Param{
				Key:   "rename-command",
				Value: "CONFIG aliasedconfigcommand",
			},
		)

		err = conf.Save(confPath)
		Ω(err).ShouldNot(HaveOccurred())

		cwd, err := os.Getwd()
		Ω(err).ShouldNot(HaveOccurred())

		aofPath = filepath.Join(cwd, "appendonly.aof")
		_, err = os.Create(aofPath)
		Ω(err).ShouldNot(HaveOccurred())

		rdbPath = filepath.Join(cwd, "dump.rdb")
		_, err = os.Create(rdbPath)
		Ω(err).ShouldNot(HaveOccurred())

		redisClient = resetter.New(defaultConfPath, confPath, fakePortChecker, commandRunner, monitExecutablePath, fakeMonit)
	})

	AfterEach(func() {
		os.Remove(aofPath)
		os.Remove(rdbPath)
	})

	Describe("#ResetRedis", func() {
		It("stops and starts redis with monit", func() {
			err := redisClient.ResetRedis()
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeMonit.StopAndWaitCallCount()).To(Equal(1))
			Expect(fakeMonit.StartAndWaitCallCount()).To(Equal(1))
		})

		It("removes the AOF file", func() {
			err := redisClient.ResetRedis()
			Ω(err).ShouldNot(HaveOccurred())

			_, err = os.Stat(aofPath)
			Ω(os.IsNotExist(err)).To(BeTrue())
		})

		It("removes the RDB file", func() {
			err := redisClient.ResetRedis()
			Ω(err).ShouldNot(HaveOccurred())

			_, err = os.Stat(rdbPath)
			Ω(os.IsNotExist(err)).To(BeTrue())
		})

		It("nukes the config file and replaces it with one containing a new password", func() {
			err := redisClient.ResetRedis()
			Ω(err).ShouldNot(HaveOccurred())

			newConfig, err := redisconf.Load(confPath)
			Ω(err).ShouldNot(HaveOccurred())

			newPassword := newConfig.Get("requirepass")
			Ω(newPassword).NotTo(BeEmpty())
			Ω(newPassword).NotTo(Equal(redisPassword))
		})

		It("does not return until redis is available again", func() {
			err := redisClient.ResetRedis()
			Ω(err).ShouldNot(HaveOccurred())

			address, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("localhost:%d", redisPort))
			Ω(err).ShouldNot(HaveOccurred())

			Ω(fakePortChecker.addressesWaitedOn).To(ConsistOf(address))
		})

		Context("when redis fails to become available again within the timeout period", func() {
			It("returns the error from the checker", func() {
				fakePortChecker.checkErr = errors.New("I timed out")
				err := redisClient.ResetRedis()
				Ω(err).Should(MatchError("I timed out"))
			})
		})

		Context("when the AOF file cannot be removed", func() {
			JustBeforeEach(func() {
				err := os.Remove(aofPath)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("returns error", func() {
				Ω(redisClient.ResetRedis()).Should(HaveOccurred())
			})
		})
	})
})
