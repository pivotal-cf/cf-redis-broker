package redisinstance

import (
	"fmt"
	"net/http"
)

type InstanceIDFinder interface {
	IDForHost(string) string
}

func NewHandler(instanceIDFinder InstanceIDFinder) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		res.Header().Add("Content-Type", "application/json")

		instanceID := instanceIDFinder.IDForHost(req.URL.Query()["host"][0])
		if instanceID == "" {
			http.Error(res, "", http.StatusNotFound)
			return
		}

		res.Write([]byte(`{"instance_id":"` + instanceID + `"}`))
	}
}

func NewIsAllocatedHandler(instanceIDFinder InstanceIDFinder) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		instanceID := instanceIDFinder.IDForHost(req.URL.Query()["host"][0])

		res.Write([]byte(fmt.Sprintf(`{"is_allocated": %t}`, instanceID != "")))
	}
}
