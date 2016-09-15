package fakes

import (
	"fmt"
	"time"

	"github.com/pivotal-cf/cf-redis-broker/redis/client"
)

type Client struct {
	ExpectedRDBPathErr error
	ExpectedRDBPath    string
	InvokedRDBPath     int

	RunBGSaveCallCount   int
	ExpectedRunGBSaveErr error

	WaitForNewSaveSinceCallCount   int
	ExpectedWaitForNewSaveSinceErr error
	PingReturns                    error

	Host string
	Port int
}

func (c *Client) Exec(command string, args ...interface{}) (interface{}, error) {
	return nil, nil
}

func (c *Client) GlobalKeyCount() (int, error) {
	return 0, nil
}

func (c *Client) Address() string {
	if c.Host == "" {
		return ""
	}
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func (c *Client) Disconnect() error {
	return nil
}

func (c *Client) WaitUntilRedisNotLoading(timeoutMilliseconds int) error {
	return nil
}

func (c *Client) EnableAOF() error {
	return nil
}

func (c *Client) LastRDBSaveTime() (int64, error) {
	return 0, nil
}

func (c *Client) InfoField(fieldName string) (string, error) {
	return "", nil
}

func (c *Client) Info() (map[string]string, error) {
	return map[string]string{}, nil
}

func (c *Client) GetConfig(key string) (string, error) {
	return "", nil
}

func (c *Client) RDBPath() (string, error) {
	c.InvokedRDBPath++
	return c.ExpectedRDBPath, c.ExpectedRDBPathErr
}

func (c *Client) RunBGSave() error {
	c.RunBGSaveCallCount++
	return c.ExpectedRunGBSaveErr
}

func (c *Client) WaitForNewSaveSince(lastSaveTime int64, timeout time.Duration) error {
	c.WaitForNewSaveSinceCallCount++
	return c.ExpectedWaitForNewSaveSinceErr
}

func (c *Client) Ping() error {
	return c.PingReturns
}

var _ client.Client = new(Client)
