package log

import (
	"fmt"
	"io"
	"os"
	"strings"

	"code.cloudfoundry.org/lager"
)

type lagerEntry struct {
	Data      map[string]interface{}
	LogLevel  int
	Message   string
	Source    string
	Timestamp string
}

type CliSink struct {
	writer      io.Writer
	minLogLevel lager.LogLevel
}

func NewCliSink(minLogLevel lager.LogLevel) lager.Sink {
	return &CliSink{
		writer:      os.Stdout,
		minLogLevel: minLogLevel,
	}
}

func (s *CliSink) Log(format lager.LogFormat) {
	if format.LogLevel < s.minLogLevel {
		return
	}

	if format.Message == "" || format.Data == nil || format.Data["event"] == "" {
		return
	}

	fmt.Fprintln(s.writer, prettify(format.Message, format.Data["event"]))
}

func prettify(message string, event interface{}) string {
	return fmt.Sprintf(
		"%15s -> %s",
		splitLagerMessage(message),
		event,
	)
}

func splitLagerMessage(message string) string {
	parts := strings.Split(message, ".")
	return parts[len(parts)-1]
}
