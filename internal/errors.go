package internal

import (
	"errors"
	"log"
	"net/http"
)

var (
	ErrTodo = errors.New("rmx: not yet implemented")

	errNoCookie        = errorResponse{status: http.StatusUnauthorized, message: "Cookie not found."}
	errSessionNotFound = errorResponse{status: http.StatusNotFound, message: "Session not found."}
	errSessionExists   = errorResponse{status: http.StatusNotFound, message: "Session already exists."}
)

func handlerError(w http.ResponseWriter, err error) {
	if err != nil {
		if httpError, ok := err.(*errorResponse); ok {
			http.Error(w, httpError.message, httpError.status)
			return
		}
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
