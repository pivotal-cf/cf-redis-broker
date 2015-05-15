package log_test

import (
	"io"
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/cf-redis-broker/log"
	"github.com/pivotal-golang/lager"
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
				sink.Log(lager.INFO, []byte("Logging to stdout"))
				os.Stdout.Close()

				output, _ := ioutil.ReadAll(stdoutReader)
				Expect(string(output)).To(Equal(""))
			})
		})

		Context("when payload has lager data with event key", func() {
			It("prints a prettified message to stdout", func() {
				message := `
					{
						"timestamp":"1431625200.765033007",
						"source":"backup",
						"message":"backup.backup_main",
						"log_level":1,
						"data":{
							"event":"Exiting",
							"exit_code":1
						}
					}
				`
				sink.Log(lager.INFO, []byte(message))
				os.Stdout.Close()

				output, _ := ioutil.ReadAll(stdoutReader)
				Expect(string(output)).To(Equal("    backup_main -> Exiting\n"))
			})
		})
	})

})
