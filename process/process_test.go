package process_test

import (
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/pivotal-cf/cf-redis-broker/process"
	"github.com/pivotal-cf/cf-redis-broker/process/processfakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("process", func() {
	Describe(".ReadPID", func() {
		Context("when the pid file exists", func() {
			var pidFilePath string

			BeforeEach(func() {
				pidFile, err := ioutil.TempFile("", "pid")
				Expect(err).ToNot(HaveOccurred())

				pidFilePath = pidFile.Name()

				_, err = pidFile.WriteString("1234")
				Expect(err).ToNot(HaveOccurred())

				err = pidFile.Close()
				Expect(err).ToNot(HaveOccurred())
			})

			It("reads the pid from the pid file", func() {
				pid, err := process.ReadPID(pidFilePath)
				Expect(err).ToNot(HaveOccurred())
				Expect(pid).To(Equal(1234))
			})

			Context("when the pid file is invalid", func() {
				BeforeEach(func() {
					err := ioutil.WriteFile(pidFilePath, []byte("bs"), os.ModePerm)
					Expect(err).ToNot(HaveOccurred())
				})

				It("returns an error", func() {
					_, err := process.ReadPID(pidFilePath)
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Context("when the pid file does not exists", func() {
			It("returns an error", func() {
				_, err := process.ReadPID("/foo/bar")
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("ProcessKiller", func() {
		Describe("Alive", func() {
			It("returns false when the process is not alive", func() {
				cmd := exec.Command("sleep", "60")
				cmd.Start()

				pid := cmd.Process.Pid
				Ω(new(process.ProcessChecker).Alive(pid)).Should(BeTrue())
				err := new(process.ProcessKiller).Kill(pid)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(new(process.ProcessChecker).Alive(pid)).Should(BeFalse())
			})

			It("returns true when the process is alive", func() {
				alive := new(process.ProcessChecker).Alive(os.Getpid())
				Ω(alive).Should(BeTrue())
			})
		})

		Describe("Kill", func() {
			It("does not return an error when the process has been killed", func() {
				cmd := exec.Command("sleep", "60")
				cmd.Start()

				pid := cmd.Process.Pid
				Ω(new(process.ProcessChecker).Alive(pid)).Should(BeTrue())
				err := new(process.ProcessKiller).Kill(pid)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(new(process.ProcessChecker).Alive(pid)).Should(BeFalse())
			})

			It("returns an error when the process is not alive", func() {
				cmd := exec.Command("sleep", "60")
				cmd.Start()

				pid := cmd.Process.Pid
				Ω(new(process.ProcessChecker).Alive(pid)).Should(BeTrue())
				err := new(process.ProcessKiller).Kill(pid)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(new(process.ProcessChecker).Alive(pid)).Should(BeFalse())

				err = new(process.ProcessKiller).Kill(pid)
				Ω(err).Should(HaveOccurred())
			})
		})
	})

	Describe("ProcessChecker", func() {
		FDescribe("IsRedisServer", func() {
			var processFinder *processfakes.FakeProcessFinder
			var finderProvider *processfakes.FakeFinderProvider

			Context("when the process is redis-server", func() {
				BeforeEach(func() {

					processFinder = new(processfakes.FakeProcessFinder)
					processFinder.NameReturns("redis-server", nil)
					finderProvider = new(processfakes.FakeFinderProvider)
					finderProvider.NewProcessReturns(processFinder, nil)

				})

				It("return true", func() {
					Expect(new(process.ProcessChecker).IsRedisServer(finderProvider, 1)).To(BeTrue())
					Expect(finderProvider.NewProcessArgsForCall(0)).To(Equal(1))
				})
			})

			Context("when the process is redis-server with details", func() {
				BeforeEach(func() {

					processFinder = new(processfakes.FakeProcessFinder)
					processFinder.NameReturns("redis-server *6879", nil)
					finderProvider = new(processfakes.FakeFinderProvider)
					finderProvider.NewProcessReturns(processFinder, nil)

				})

				It("return true", func() {
					Expect(new(process.ProcessChecker).IsRedisServer(finderProvider, 1)).To(BeTrue())
					Expect(finderProvider.NewProcessArgsForCall(0)).To(Equal(1))
				})
			})

			Context("when the process is not redis-server", func() {
				BeforeEach(func() {

					processFinder = new(processfakes.FakeProcessFinder)
					processFinder.NameReturns("fail", nil)
					finderProvider = new(processfakes.FakeFinderProvider)
					finderProvider.NewProcessReturns(processFinder, nil)

				})

				It("return false", func() {
					Expect(new(process.ProcessChecker).IsRedisServer(finderProvider, 1)).To(BeFalse())
					Expect(finderProvider.NewProcessArgsForCall(0)).To(Equal(1))
				})
			})
		})
	})
})
