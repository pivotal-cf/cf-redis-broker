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

func (fakePortChecker *fakeChecker) Check(address *net.TCPAddr, timeout time.Duration) error {
	fakePortChecker.addressesWaitedOn = append(fakePortChecker.addressesWaitedOn, address)
	return fakePortChecker.checkErr
}

type fakeShell struct {
	commandsRan []*exec.Cmd
}

func (shell *fakeShell) Run(command *exec.Cmd) ([]byte, error) {
	shell.commandsRan = append(shell.commandsRan, command)
	return []byte{}, nil
}

var _ = Describe("Client", func() {

	var redisClient *resetter.Resetter
	var fakePortChecker *fakeChecker
	var shell *fakeShell
	var monitExecutable = "/path/to/monit"
	var aofPath string
	var rdbPath string
	var redisPort int
	var redisPassword string
	var confPath string
	var defaultConfPath string
	var conf redisconf.Conf

	BeforeEach(func() {
		shell = new(fakeShell)
		fakePortChecker = new(fakeChecker)
		redisPassword = "somepassword"
		dir, _ := ioutil.TempDir("", "redisconf-test")
		defaultConfPath = filepath.Join(dir, "redis.conf-default")
		confPath = filepath.Join(dir, "redis.conf")
		err := redisconf.New(
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
		redisClient = resetter.New(defaultConfPath, confPath, fakePortChecker, shell, monitExecutable)
	})

	JustBeforeEach(func() {
		err := conf.Save(confPath)
		Ω(err).ShouldNot(HaveOccurred())

		cwd, err := os.Getwd()
		Ω(err).ShouldNot(HaveOccurred())
		aofPath = filepath.Join(cwd, "appendonly.aof")
		_, err = os.Create(aofPath)
		Ω(err).ShouldNot(HaveOccurred())
		rdbPath = filepath.Join(cwd, "dump.rdb")
		_, err = os.Create(rdbPath)
		Ω(err).ShouldNot(HaveOccurred())
	})

	AfterEach(func() {
		os.Remove(aofPath)
		os.Remove(rdbPath)
	})

	Describe("#DeleteAllData", func() {
		It("stops and starts redis with monit", func() {
			err := redisClient.DeleteAllData()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(shell.commandsRan).To(HaveLen(2))
			Ω(shell.commandsRan[0].Path).To(Equal(monitExecutable))
			Ω(shell.commandsRan[0].Args).To(ConsistOf(monitExecutable, "stop", "redis"))
			Ω(shell.commandsRan[1].Path).To(Equal(monitExecutable))
			Ω(shell.commandsRan[1].Args).To(ConsistOf(monitExecutable, "start", "redis"))
		})

		It("removes the AOF file", func() {
			err := redisClient.DeleteAllData()
			Ω(err).ShouldNot(HaveOccurred())

			_, err = os.Stat(aofPath)
			Ω(os.IsNotExist(err)).To(BeTrue())
		})

		It("removes the RDB file", func() {
			err := redisClient.DeleteAllData()
			Ω(err).ShouldNot(HaveOccurred())

			_, err = os.Stat(rdbPath)
			Ω(os.IsNotExist(err)).To(BeTrue())
		})

		It("nukes the config file and replaces it with one containing a new password", func() {
			err := redisClient.DeleteAllData()
			Ω(err).ShouldNot(HaveOccurred())

			newConfig, err := redisconf.Load(confPath)
			Ω(err).ShouldNot(HaveOccurred())

			newPassword := newConfig.Get("requirepass")
			Ω(newPassword).NotTo(BeEmpty())
			Ω(newPassword).NotTo(Equal(redisPassword))

			Ω(newConfig.HasKey("appendonly")).Should(BeFalse())
		})

		It("does not return until redis is available again", func() {
			err := redisClient.DeleteAllData()
			Ω(err).ShouldNot(HaveOccurred())
			address, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("localhost:%d", redisPort))
			Ω(err).ShouldNot(HaveOccurred())
			Ω(fakePortChecker.addressesWaitedOn).To(ConsistOf(address))
		})

		Context("when redis fails to become available again within the timeout period", func() {
			It("returns the error from the checker", func() {
				fakePortChecker.checkErr = errors.New("I timed out")
				err := redisClient.DeleteAllData()
				Ω(err).Should(MatchError("I timed out"))
			})
		})

		Context("when the AOF file cannot be found", func() {

			JustBeforeEach(func() {
				err := os.Remove(aofPath)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("returns error", func() {
				Ω(redisClient.DeleteAllData()).Should(HaveOccurred())
			})
		})
	})
})
