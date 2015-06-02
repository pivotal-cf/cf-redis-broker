package instance

type IDProvider interface {
	InstanceID(redisConfigPath, nodeIP string) (string, error)
}
