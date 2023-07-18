package redis_test

import (
	"encoding/json"
	"io/ioutil"
	"net"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf/cf-redis-broker/redis"
)

type Statefile struct {
	AvailableInstances []*redis.Instance   `json:"available_instances"`
	AllocatedInstances []*redis.Instance   `json:"allocated_instances"`
	InstanceBindings   map[string][]string `json:"instance_bindings"`
}

func TestRedis(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Redis Suite")
}

func getStatefileContents(path string) Statefile {
	statefileBytes, _ := ioutil.ReadFile(path)
	statefileContents := Statefile{}

	err := json.Unmarshal(statefileBytes, &statefileContents)
	Expect(err).ToNot(HaveOccurred())
	return statefileContents
}

func putStatefileContents(path string, file Statefile) {
	statefileBytes, err := json.Marshal(&file)
	Expect(err).ToNot(HaveOccurred())

	err = ioutil.WriteFile(path, statefileBytes, 0644)
	Expect(err).ToNot(HaveOccurred())
}

func isListeningChecker(uri string) func() bool {
	return func() bool {
		return isListening(uri)
	}
}

func isListening(uri string) bool {
	address, err := net.ResolveTCPAddr("tcp", uri)
	Expect(err).ToNot(HaveOccurred())

	_, err = net.DialTCP("tcp", nil, address)
	return err == nil
}
