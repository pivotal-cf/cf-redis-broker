package debug

import (
	"encoding/json"
	"net/http"

	"github.com/pivotal-cf/cf-redis-broker/redis"
)

func NewHandler(repo *redis.RemoteRepository) func(http.ResponseWriter, *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		res.Header().Add("Content-Type", "application/json")

		debugInfoBytes, err := buildDebugInfoBytes(repo)

		if err != nil {
			res.Write([]byte(http.StatusText(http.StatusInternalServerError)))
			res.WriteHeader(http.StatusInternalServerError)
		} else {
			res.Write(debugInfoBytes)
		}
	}
}

type Binding struct {
	ID string `json:"id"`
}

type Cluster struct {
	ID       string
	Hosts    []string  `json:"hosts"`
	Bindings []Binding `json:"bindings"`
}

type Pool struct {
	Count    int        `json:"count"`
	Clusters [][]string `json:"clusters"`
}

type Allocated struct {
	Count    int       `json:"count"`
	Clusters []Cluster `json:"clusters"`
}

type Info struct {
	Pool      Pool      `json:"pool"`
	Allocated Allocated `json:"allocated"`
}

func buildDebugInfoBytes(repo *redis.RemoteRepository) ([]byte, error) {
	allocatedInfo, err := getAllocatedInfo(repo)
	if err != nil {
		return nil, err
	}

	return json.Marshal(&Info{
		Pool:      getPoolInfo(repo),
		Allocated: allocatedInfo,
	})
}

func getPoolInfo(repo *redis.RemoteRepository) Pool {
	pool := Pool{}

	availableNodes := []string{}
	for _, instance := range repo.AvailableInstances() {
		availableNodes = append(availableNodes, instance.Host)
	}

	pool.Count = len(availableNodes)

	for _, node := range availableNodes {
		cluster := []string{node}
		pool.Clusters = append(pool.Clusters, cluster)
	}

	return pool
}

func getAllocatedInfo(repo *redis.RemoteRepository) (Allocated, error) {
	allocated := Allocated{}

	allocatedInstances, err := repo.AllInstances()
	if err != nil {
		return allocated, err
	}

	for _, instance := range allocatedInstances {
		bindingIDs, err := repo.BindingsForInstance(instance.ID)
		if err != nil {
			return allocated, err
		}

		c := Cluster{
			ID:    instance.ID,
			Hosts: []string{instance.Host},
		}

		for _, id := range bindingIDs {
			c.Bindings = append(c.Bindings, Binding{ID: id})
		}

		allocated.Clusters = append(allocated.Clusters, c)
	}

	allocated.Count = len(allocatedInstances)

	return allocated, nil
}
