package plan

type IDProvider interface {
	InstanceID(redisConfigPath, nodeIP string) (string, error)
}
