package backup

import (
	"fmt"
	"os"

	"github.com/pivotal-cf/cf-redis-broker/recovery/task"
	"github.com/pivotal-golang/lager"
)

type cleanup struct {
	originalRdbPath string
	renamedRdbPath  string
	logger          lager.Logger
	remove          remover
	rename          renamer
}

type renamer func(string, string) error
type remover func(string) error

type cleanupOption func(*cleanup)

func InjectRemover(r remover) cleanupOption {
	return func(c *cleanup) {
		c.remove = r
	}
}

func InjectRenamer(r renamer) cleanupOption {
	return func(c *cleanup) {
		c.rename = r
	}
}

func NewCleanup(originalRdbPath, renamedRdbPath string, logger lager.Logger, options ...cleanupOption) task.Task {
	c := &cleanup{
		remove:          os.Remove,
		rename:          os.Rename,
		originalRdbPath: originalRdbPath,
		renamedRdbPath:  renamedRdbPath,
		logger:          logger,
	}

	for _, option := range options {
		option(c)
	}

	return c
}

func (c *cleanup) Name() string {
	return "cleanup"
}

func (c *cleanup) Run(artifact task.Artifact) (task.Artifact, error) {
	logData := lager.Data{
		"original_path": c.originalRdbPath,
		"renamed_path":  c.renamedRdbPath,
	}

	c.logInfo("", "starting", logData)

	if !fileExists(c.originalRdbPath) && fileExists(c.renamedRdbPath) {
		if err := c.moveDump(); err != nil {
			return artifact, err
		}
	}

	if fileExists(c.renamedRdbPath) {
		if err := c.removeRenamedDump(); err != nil {
			return artifact, err
		}
	}

	c.logInfo("", "done", logData)

	return artifact, nil
}

func (c *cleanup) moveDump() error {
	logData := lager.Data{
		"old_path": c.renamedRdbPath,
		"new_path": c.originalRdbPath,
	}

	c.logInfo("rename", "starting", logData)

	if err := c.rename(c.renamedRdbPath, c.originalRdbPath); err != nil {
		c.logError("rename", err, logData)
		return err
	}

	c.logInfo("rename", "done", logData)

	return nil
}

func (c *cleanup) removeRenamedDump() error {
	logData := lager.Data{
		"path": c.renamedRdbPath,
	}

	c.logInfo("remove", "starting", logData)

	if err := c.remove(c.renamedRdbPath); err != nil {
		c.logError("remove", err, logData)
		return err
	}

	c.logInfo("remove", "done", logData)

	return nil
}

func (c *cleanup) logInfo(subAction, event string, data lager.Data) {
	data["event"] = event

	c.logger.Info(
		c.logAction(subAction),
		data,
	)
}

func (c *cleanup) logError(subAction string, err error, data lager.Data) {
	data["event"] = "failed"

	c.logger.Error(
		c.logAction(subAction),
		err,
		data,
	)
}

func (c *cleanup) logAction(subAction string) string {
	action := c.Name()
	if subAction != "" {
		action = fmt.Sprintf("%s.%s", action, subAction)
	}

	return action
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}
