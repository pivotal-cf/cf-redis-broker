package redisinstance

import (
	"encoding/json"
	"net/http"
)

type InstanceIDFinder interface {
	IDForHost(string) string
}

type Response struct {
	InstanceID string `json:"instance_id"`
}

func NewHandler(instanceIDFinder InstanceIDFinder) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		res.Header().Add("Content-Type", "application/json")

		host := req.URL.Query()["host"]
		if len(host) == 0 {
			http.Error(res, "", http.StatusBadRequest)
			return
		}

		instanceID := instanceIDFinder.IDForHost(host[0])
		if instanceID == "" {
			http.Error(res, "", http.StatusNotFound)
			return
		}

		payload, err := json.Marshal(Response{instanceID})
		if err != nil {
			http.Error(res, "", http.StatusInternalServerError)
		}

		res.Write(payload)
	}
}
