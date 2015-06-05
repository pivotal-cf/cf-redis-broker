package id

type InstanceIDLocator interface {
	LocateID(redisConfigPath, nodeIP string) (string, error)
}
