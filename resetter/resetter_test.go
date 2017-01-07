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
		Expect(err).NotTo(HaveOccurred())
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
		Expect(err).NotTo(HaveOccurred())

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
		Expect(err).NotTo(HaveOccurred())

		cwd, err := os.Getwd()
		Expect(err).NotTo(HaveOccurred())

		aofPath = filepath.Join(cwd, "appendonly.aof")
		_, err = os.Create(aofPath)
		Expect(err).NotTo(HaveOccurred())

		rdbPath = filepath.Join(cwd, "dump.rdb")
		_, err = os.Create(rdbPath)
		Expect(err).NotTo(HaveOccurred())

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
			Expect(err).NotTo(HaveOccurred())

			_, err = os.Stat(aofPath)
			Expect(os.IsNotExist(err)).To(BeTrue())
		})

		It("removes the RDB file", func() {
			err := redisClient.ResetRedis()
			Expect(err).NotTo(HaveOccurred())

			_, err = os.Stat(rdbPath)
			Expect(os.IsNotExist(err)).To(BeTrue())
		})

		It("nukes the config file and replaces it with one containing a new password", func() {
			err := redisClient.ResetRedis()
			Expect(err).NotTo(HaveOccurred())

			newConfig, err := redisconf.Load(confPath)
			Expect(err).NotTo(HaveOccurred())

			newPassword := newConfig.Get("requirepass")
			Expect(newPassword).NotTo(BeEmpty())
			Expect(newPassword).NotTo(Equal(redisPassword))
		})

		It("does not return until redis is available again", func() {
			err := redisClient.ResetRedis()
			Expect(err).NotTo(HaveOccurred())

			address, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("localhost:%d", redisPort))
			Expect(err).NotTo(HaveOccurred())

			Expect(fakePortChecker.addressesWaitedOn).To(ConsistOf(address))
		})

		Context("when redis fails to become available again within the timeout period", func() {
			It("returns the error from the checker", func() {
				fakePortChecker.checkErr = errors.New("I timed out")
				err := redisClient.ResetRedis()
				Expect(err).To(MatchError("I timed out"))
			})
		})

		Context("when the AOF file cannot be removed", func() {
			JustBeforeEach(func() {
				err := os.Remove(aofPath)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns error", func() {
				Expect(redisClient.ResetRedis()).To(HaveOccurred())
			})
		})
	})
})
