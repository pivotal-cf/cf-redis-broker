package fakes

import (
	"fmt"
	"time"
)

type Client struct {
	ExpectedCreateSnapshotErr error
	InvokedCreateSnapshot     []time.Duration

	ExpectedRDBPathErr error
	ExpectedRDBPath    string
	InvokedRDBPath     int

	Host string
	Port int
}

func (c *Client) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func (c *Client) CreateSnapshot(timeout time.Duration) error {
	if c.InvokedCreateSnapshot == nil {
		c.InvokedCreateSnapshot = []time.Duration{}
	}
	c.InvokedCreateSnapshot = append(c.InvokedCreateSnapshot, timeout)
	return c.ExpectedCreateSnapshotErr
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

func (c *Client) GetConfig(key string) (string, error) {
	return "", nil
}

func (c *Client) RDBPath() (string, error) {
	c.InvokedRDBPath++
	return c.ExpectedRDBPath, c.ExpectedRDBPathErr
}
