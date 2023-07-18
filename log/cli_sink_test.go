package log_test

import (
	"io"
	"io/ioutil"
	"os"

	"code.cloudfoundry.org/lager/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf/cf-redis-broker/log"
)

var _ = Describe("CliSink", func() {
	var (
		stdoutReader   io.Reader
		originalStdout *os.File
		sink           lager.Sink
	)

	BeforeEach(func() {
		originalStdout = os.Stdout
		stdoutReader, os.Stdout, _ = os.Pipe()
		sink = log.NewCliSink(lager.INFO)
	})

	AfterEach(func() {
		os.Stdout = originalStdout
	})

	Describe(".Log", func() {
		Context("when the payload doesn't have lager data with event key", func() {
			It("does not write to stdout", func() {
				sink.Log(lager.LogFormat{
					LogLevel: lager.INFO,
					Message:  "Logging to stdout",
				})

				os.Stdout.Close()

				output, _ := ioutil.ReadAll(stdoutReader)
				Expect(string(output)).To(Equal(""))
			})
		})

		Context("when the payload does have lager data with event key set to empty string", func() {
			It("does not write to stdout", func() {
				sink.Log(lager.LogFormat{
					LogLevel: lager.INFO,
					Message:  "Logging to stdout",
					Data:     lager.Data{"event": ""},
				})

				os.Stdout.Close()

				output, _ := ioutil.ReadAll(stdoutReader)
				Expect(string(output)).To(Equal(""))
			})
		})

		Context("when payload has lager data with event key", func() {
			It("prints a prettified message to stdout", func() {
				sink.Log(lager.LogFormat{
					Timestamp: "1431625200.765033007",
					Source:    "backup",
					Message:   "backup.backup_main",
					LogLevel:  lager.INFO,
					Data: lager.Data{
						"event":     "Exiting",
						"exit_code": 1,
					},
				})

				os.Stdout.Close()

				output, _ := ioutil.ReadAll(stdoutReader)
				Expect(string(output)).To(Equal("    backup_main -> Exiting\n"))
			})
		})
	})

})
