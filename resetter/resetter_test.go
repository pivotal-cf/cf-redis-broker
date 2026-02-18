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
	"github.com/pivotal-cf/redisutils/redis"

	. "github.com/onsi/ginkgo/v2"
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
		redisFake       *redis.Fake
		connFake        *redis.ConnFake

		redisPassword = "somepassword"
	)

	BeforeEach(func() {
		redisFake = new(redis.Fake)
		connFake = new(redis.ConnFake)
		redisFake.DialReturns(connFake, nil)
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
		redisClient.redis = redisFake
		redisClient.Monit = fakeMonit
	})

	AfterEach(func() {
		os.Remove(aofPath)
		os.Remove(rdbPath)
	})

	Describe("#ResetRedis", func() {
		var resetErr error

		JustBeforeEach(func() {
			resetErr = redisClient.ResetRedis()
		})

		It("does not return an error", func() {
			Expect(resetErr).NotTo(HaveOccurred())
		})

		It("invokes script kill", func() {
			Expect(connFake.DoCallCount()).To(BeNumerically(">", 1))
			response, args := connFake.DoArgsForCall(1)
			Expect(response).To(Equal("SCRIPT"))
			Expect(args).To(Equal([]interface{}{"KILL"}))
		})

		It("closes the redis connection", func() {
			Expect(connFake.CloseCallCount()).To(Equal(1))
		})

		Context("when auth returns an error", func() {
			authErr := errors.New("failed to authenticate")

			BeforeEach(func() {
				connFake.DoReturns(nil, authErr)
			})

			It("returns the error", func() {
				Expect(resetErr).To(MatchError(authErr))
			})

			It("closes the redis connection", func() {
				Expect(connFake.CloseCallCount()).To(Equal(1))
			})
		})

		Context("when script kill returns an error", func() {
			scriptKillErr := errors.New("Failed to script kill")

			BeforeEach(func() {
				connFake.DoStub = doReturns([]interfaceAndErr{
					{nil, nil},
					{nil, scriptKillErr},
				}).sequentially
			})

			It("returns the error", func() {
				expected := fmt.Errorf("failed to kill redis script: %s", scriptKillErr.Error())
				Expect(resetErr).To(MatchError(expected))
			})
		})

		Context("when there's no script running", func() {
			scriptKillErr := errors.New("No scripts in execution right now")

			BeforeEach(func() {
				connFake.DoStub = doReturns([]interfaceAndErr{
					{nil, nil},
					{nil, scriptKillErr},
				}).sequentially
			})

			It("does not return an error", func() {
				Expect(resetErr).NotTo(HaveOccurred())
			})
		})

		It("stops and starts redis with monit", func() {
			Expect(resetErr).NotTo(HaveOccurred())
			Expect(fakeMonit.StopAndWaitCallCount()).To(Equal(1))
			Expect(fakeMonit.StartAndWaitCallCount()).To(Equal(1))
		})

		Context("when `monit start` fails", func() {
			monitStartError := errors.New("Monit has failed to start")

			BeforeEach(func() {
				fakeMonit.StartAndWaitReturns(monitStartError)
			})

			It("monit start returns an error", func() {
				Expect(resetErr).To(MatchError(monitStartError))
			})
		})

		Context("when `monit stop` fails", func() {
			monitStopError := errors.New("Monit has failed to stop")

			BeforeEach(func() {
				fakeMonit.StopAndWaitReturns(monitStopError)
			})

			It("monit stop returns an error", func() {
				Expect(resetErr).To(MatchError(monitStopError))
			})
		})

		It("removes the AOF file", func() {
			Expect(aofPath).NotTo(BeAnExistingFile())
		})

		It("removes the RDB file", func() {
			Expect(rdbPath).NotTo(BeAnExistingFile())
		})

		It("nukes the config file and replaces it with one containing a new password", func() {
			newConfig, err := redisconf.Load(confPath)
			Expect(err).NotTo(HaveOccurred())

			newPassword := newConfig.Get("requirepass")
			Expect(newPassword).NotTo(BeEmpty())
			Expect(newPassword).NotTo(Equal(redisPassword))
		})

		It("does not return until redis is available again", func() {
			address, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("127.0.0.1:%d", redisPort))
			Expect(err).NotTo(HaveOccurred())

			Expect(fakePortChecker.addressesWaitedOn).To(ConsistOf(address))
		})

		Context("when redis fails to become available again within the timeout period", func() {
			checkErr := errors.New("I timed out")

			BeforeEach(func() {
				fakePortChecker.checkErr = checkErr
			})

			It("returns the error from the checker", func() {
				Expect(resetErr).To(MatchError(checkErr))
			})
		})

		Context("when the AOF file cannot be removed", func() {
			BeforeEach(func() {
				err := os.Remove(aofPath)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns error", func() {
				Expect(resetErr).To(HaveOccurred())
			})
		})
	})
})
