package http

import (
	"encoding/json"
	"net/http"
	"strings"
)

func Respond(rw http.ResponseWriter, r *http.Request, data interface{}, status int) {
	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	rw.WriteHeader(status)
	if data != nil {
		err := json.NewEncoder(rw).Encode(data)
		if err != nil {
			http.Error(rw, "Could not encode in json", status)
		}
	}
}

func Created(rw http.ResponseWriter, r *http.Request, id string) {
	path := r.URL.Path
	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}
	rw.Header().Add("Location", "//"+r.Host+path+id)
	Respond(rw, r, nil, http.StatusCreated)
}

func Decode(rw http.ResponseWriter, r *http.Request, data interface{}) (err error) {
	return json.NewDecoder(r.Body).Decode(data)
}

func Echo(message string) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) { Respond(rw, r, message, http.StatusOK) }
}

func FileServer(prefix, dirname string) http.Handler {
	return http.StripPrefix(prefix, http.FileServer(http.Dir(dirname)))
}
