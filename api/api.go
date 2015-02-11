package api

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pivotal-cf/cf-redis-broker/credentials"
)

type redisResetter interface {
	ResetRedis() error
}

type credentialsParserFunc func(string) (credentials.Credentials, error)

func New(resetter redisResetter, configPath string, parseCredentials credentialsParserFunc) http.Handler {
	router := mux.NewRouter()

	router.Path("/").
		Methods("DELETE").
		HandlerFunc(resetHandler(resetter))

	router.Path("/").
		Methods("GET").
		HandlerFunc(credentialsHandler(configPath, parseCredentials))

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

func credentialsHandler(configPath string, parseCredentials credentialsParserFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		credentials, err := parseCredentials(configPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		encoder := json.NewEncoder(w)
		encoder.Encode(credentials)
	}
}
