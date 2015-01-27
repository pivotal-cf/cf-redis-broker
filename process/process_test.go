package process_test

import (
	"os"
	"os/exec"

	"github.com/pivotal-cf/cf-redis-broker/process"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("process", func() {
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
