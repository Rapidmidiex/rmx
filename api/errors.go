package api

import (
	"log"
	"net/http"
)

var (
	errNoCookie            = errorResponse{status: http.StatusUnauthorized, message: "Cookie not found."}
	errUsernameAlreadyUsed = errorResponse{status: http.StatusForbidden, message: "This username is already used."}
	errSessionNotFound     = errorResponse{status: http.StatusNotFound, message: "Session not found."}
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
