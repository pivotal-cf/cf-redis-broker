package glager

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"
	"github.com/pivotal-golang/lager"
)

type logEntry lager.LogFormat

type logEntries []logEntry

type logEntryData lager.Data

type option func(*logEntry)

type logMatcher struct {
	actual   logEntries
	expected logEntries
}

func ContainSequence(expectedSequence ...logEntry) types.GomegaMatcher {
	return &logMatcher{
		expected: expectedSequence,
	}
}

func Info(options ...option) logEntry {
	return Entry(lager.INFO, options...)
}

func Debug(options ...option) logEntry {
	return Entry(lager.DEBUG, options...)
}

func Error(err error, options ...option) logEntry {
	if err == nil {
		err = errors.New("")
	}

	options = append(options, Data("error", err.Error()))
	return Entry(lager.ERROR, options...)
}

func Fatal(options ...option) logEntry {
	return Entry(lager.FATAL, options...)
}

func Entry(logLevel lager.LogLevel, options ...option) logEntry {
	entry := logEntry(lager.LogFormat{
		LogLevel: logLevel,
		Data:     lager.Data{},
	})

	for _, option := range options {
		option(&entry)
	}

	return entry
}

func Message(msg string) option {
	return func(e *logEntry) {
		e.Message = msg
	}
}

func Action(action string) option {
	return Message(action)
}

func Source(src string) option {
	return func(e *logEntry) {
		e.Source = src
	}
}

func Timestamp(time.Time) option {
	ts := fmt.Sprintf("%.9f", float64(time.Now().UnixNano())/1e9)
	return func(e *logEntry) {
		e.Timestamp = ts
	}
}

func Data(kv ...string) option {
	if len(kv)%2 == 1 {
		kv = append(kv, "")
	}

	return func(e *logEntry) {
		for i := 0; i < len(kv); i += 2 {
			e.Data[kv[i]] = kv[i+1]
		}
	}
}

func (lm *logMatcher) Match(actual interface{}) (success bool, err error) {
	reader, ok := actual.(io.Reader)
	if !ok {
		return false, fmt.Errorf("Contains must be passed an io.Reader. Got:\n%s", format.Object(actual, 1))
	}

	decoder := json.NewDecoder(reader)

	lm.actual = logEntries{}

	for {
		var entry logEntry
		if err := decoder.Decode(&entry); err == io.EOF {
			break
		} else if err != nil {
			return false, err
		}
		lm.actual = append(lm.actual, entry)
	}

	actualEntries := lm.actual

	for _, expected := range lm.expected {
		i, found := actualEntries.indexOf(expected)

		if !found {
			return false, nil
		}

		actualEntries = actualEntries[i+1:]
	}

	return true, nil
}

func (lm *logMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf(
		"Expected\n\t%s\nto contain log sequence \n\t%s",
		format.Object(lm.actual, 0),
		format.Object(lm.expected, 0),
	)
}

func (lm *logMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf(
		"Expected\n\t%s\nnot to contain log sequence \n\t%s",
		format.Object(lm.actual, 0),
		format.Object(lm.expected, 0),
	)
}

func (entry logEntry) LogData() logEntryData {
	return logEntryData(entry.Data)
}

func (actual logEntry) contains(expected logEntry) bool {
	if expected.Source != "" && actual.Source != expected.Source {
		return false
	}

	if expected.Message != "" && actual.Message != expected.Message {
		return false
	}

	if actual.LogLevel != expected.LogLevel {
		return false
	}

	if expected.Timestamp != "" && actual.Timestamp != expected.Timestamp {
		return false
	}

	if !actual.LogData().contains(expected.LogData()) {
		return false
	}

	return true
}

func (actual logEntryData) contains(expected logEntryData) bool {
	for k, v := range expected {
		actualValue, found := actual[k]
		if !found || v != actualValue {
			return false
		}
	}
	return true
}

func (entries logEntries) indexOf(entry logEntry) (int, bool) {
	for i, actual := range entries {
		if actual.contains(entry) {
			return i, true
		}
	}
	return 0, false
}
