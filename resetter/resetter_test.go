package resetter

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/pivotal-cf/cf-redis-broker/redisconf"
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

var _ = Describe("Client", func() {
	var (
		redisClient     *Resetter
		fakePortChecker *fakeChecker
		aofPath         string
		rdbPath         string
		redisPort       int
		confPath        string
		defaultConfPath string
		conf            redisconf.Conf
		fakeMonit       *monitFakes.FakeMonit

		redisPassword = "somepassword"
	)

	BeforeEach(func() {
		fakeMonit = new(monitFakes.FakeMonit)
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

		redisClient = New(defaultConfPath, confPath, fakePortChecker)
		redisClient.Monit = fakeMonit
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
