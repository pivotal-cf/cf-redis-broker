package fakes

type Client struct {
	ExpectedCreateSnapshotErr error
	CreateSnapshotCalls       int
}

func (c *Client) CreateSnapshot(int) error {
	c.CreateSnapshotCalls++
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
