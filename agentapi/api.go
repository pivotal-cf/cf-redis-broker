package agentapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/pivotal-cf/cf-redis-broker/redis/client"
	"github.com/pivotal-cf/cf-redis-broker/redisconf"
)

type KeycountResponse struct {
	Keycount int `json:"key_count"`
}

type redisResetter interface {
	ResetRedis() error
}

func New(resetter redisResetter, configPath string) http.Handler {
	router := mux.NewRouter()

	router.Path("/").
		Methods("DELETE").
		HandlerFunc(resetHandler(resetter))

	router.Path("/").
		Methods("GET").
		HandlerFunc(credentialsHandler(configPath))

	router.Path("/keycount").
		Methods("GET").
		HandlerFunc(keyCountHandler(configPath))

	return router
}

func resetHandler(resetter redisResetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := resetter.ResetRedis()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func credentialsHandler(configPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conf, err := redisconf.Load(configPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		port, err := strconv.Atoi(conf.Get("port"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		password := conf.Get("requirepass")

		credentials := struct {
			Port     int    `json:"port"`
			Password string `json:"password"`
		}{
			Port:     port,
			Password: password,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		encoder := json.NewEncoder(w)
		encoder.Encode(credentials)
	}
}

func keyCountHandler(configPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conf, err := redisconf.Load(configPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		port, err := strconv.Atoi(conf.Get("port"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		redis, err := client.Connect(
			client.Port(port),
			client.Password(conf.Password()),
		)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		count, err := redis.GlobalKeyCount()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		result := &KeycountResponse{
			Keycount: count,
		}

		if err := json.NewEncoder(w).Encode(result); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
