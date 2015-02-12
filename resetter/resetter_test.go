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
	commandsRan   []*exec.Cmd
	outputStrings []string
}

func (commandRunner *fakeRunner) Run(command *exec.Cmd) ([]byte, error) {
	commandRunner.commandsRan = append(commandRunner.commandsRan, command)
	commandIndex := len(commandRunner.commandsRan) - 1
	if len(commandRunner.outputStrings) <= commandIndex {
		return []byte("Process 'redis' running"), nil
	}
	return []byte(commandRunner.outputStrings[commandIndex]), nil
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

		monitExecutablePath = "/path/to/monit"
		redisPassword       = "somepassword"
	)

	BeforeEach(func() {
		commandRunner = new(fakeRunner)
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

		redisClient = resetter.New(defaultConfPath, confPath, fakePortChecker, commandRunner, monitExecutablePath)
	})

	AfterEach(func() {
		os.Remove(aofPath)
		os.Remove(rdbPath)
	})

	Describe("#ResetRedis", func() {
		It("stops and starts redis with monit", func() {
			commandRunner.outputStrings = []string{
				"",
				"",
				`The Monit daemon 5.2.4 uptime: 23m

Process 'redis-agent'               running
Process 'redis'                     something_wierd_state
Process 'syslog-configurator'       running
System 'system_d289e4bf-dc4b-4369-a7a7-a45e71319fe0' running`,
				`The Monit daemon 5.2.4 uptime: 23m

Process 'redis-agent'               running
Process 'redis'                     running
Process 'syslog-configurator'       running
System 'system_d289e4bf-dc4b-4369-a7a7-a45e71319fe0' running`,
			}
			err := redisClient.ResetRedis()
			Ω(err).ShouldNot(HaveOccurred())

			Ω(len(commandRunner.commandsRan)).To(Equal(4))

			Ω(commandRunner.commandsRan[0].Args).To(Equal(
				[]string{monitExecutablePath, "stop", "redis"},
			))
			Ω(commandRunner.commandsRan[1].Args).To(Equal(
				[]string{monitExecutablePath, "start", "redis"},
			))
			// initializing
			Ω(commandRunner.commandsRan[2].Args).To(Equal(
				[]string{monitExecutablePath, "summary"},
			))
			// running
			Ω(commandRunner.commandsRan[3].Args).To(Equal(
				[]string{monitExecutablePath, "summary"},
			))
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
