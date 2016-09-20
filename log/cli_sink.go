package log

import (
	"encoding/json"
	"errors"
	"fmt"
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
	writer lager.Sink
}

func NewCliSink(minLogLevel lager.LogLevel) lager.Sink {
	return &CliSink{
		writer: lager.NewWriterSink(os.Stdout, minLogLevel),
	}
}

func (s *CliSink) Log(level lager.LogLevel, payload []byte) {
	msg, err := prettify(payload)
	if err == nil {
		s.writer.Log(level, msg)
	}
}

func prettify(message []byte) ([]byte, error) {
	var entry lagerEntry

	err := json.Unmarshal(message, &entry)
	if err != nil || entry.Message == "" || entry.Data["event"] == "" {
		return []byte{}, errors.New("Cannot pretiffy message")
	}

	return []byte(fmt.Sprintf(
		"%15s -> %s",
		splitLagerMessage(entry.Message),
		entry.Data["event"]),
	), nil
}

func splitLagerMessage(message string) string {
	parts := strings.Split(message, ".")
	return parts[len(parts)-1]
}
