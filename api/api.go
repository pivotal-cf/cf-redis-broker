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
		HandlerFunc(deleteHandler(resetter))

	router.Path("/").
		Methods("GET").
		HandlerFunc(getHandler(configPath, parseCredentials))

	return router
}

func deleteHandler(resetter redisResetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		err := resetter.ResetRedis()

		if err != nil {
			writeError(err, http.StatusServiceUnavailable, w)
			return
		}

		writeNothing(w, http.StatusOK)
	}
}

func getHandler(configPath string, parseCredentials credentialsParserFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		credentials, err := parseCredentials(configPath)
		if err != nil {
			writeError(err, http.StatusInternalServerError, w)
			return
		}
		writeJSON(credentials, http.StatusOK, w)
	}
}

func writeJSON(js interface{}, status int, w http.ResponseWriter) {
	bytes, err := json.Marshal(js)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(status)
	w.Write(bytes)
}

func writeNothing(w http.ResponseWriter, status int) {
	w.WriteHeader(status)

	_, err := w.Write([]byte("{}"))
	if err != nil {
		writeError(err, http.StatusInternalServerError, w)
		return
	}
}

func writeError(err error, status int, w http.ResponseWriter) {
	writeJSON(Error{Err: err.Error()}, status, w)
}

type Error struct {
	Err string `json:"error"`
}

func (err Error) Error() string {
	return err.Err
}
