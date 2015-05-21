package fakes

type Client struct {
	ExpectedCreateSnapshotErr error
	InvokedCreateSnapshot     []int

	ExpectedRDBPathErr error
	ExpectedRDBPath    string
	InvokedRDBPath     int
}

func (c *Client) CreateSnapshot(timeout int) error {
	if c.InvokedCreateSnapshot == nil {
		c.InvokedCreateSnapshot = []int{}
	}
	c.InvokedCreateSnapshot = append(c.InvokedCreateSnapshot, timeout)
	return c.ExpectedCreateSnapshotErr
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
