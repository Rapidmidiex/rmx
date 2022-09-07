package api

import (
	"log"
	"net/http"
)

var (
	errInvalidUpdateInfo   = errorResponse{status: http.StatusBadRequest, message: "Invalid Username"}
	errInvalidRegisterInfo = errorResponse{status: http.StatusBadRequest, message: "Invalid Register Info"}
	errUserAlreadyExists   = errorResponse{status: http.StatusForbidden, message: "User with the same email already exists."}
	errUserNotFound        = errorResponse{status: http.StatusNotFound, message: "User not found."}
	errBadPassword         = errorResponse{status: http.StatusBadRequest, message: "Password must contain at least 8 characters and one number"}
	errInvalidEmail        = errorResponse{status: http.StatusBadRequest, message: "Invalid Email"}
	errNoCookie            = errorResponse{status: http.StatusUnauthorized, message: "Cookie not found."}
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
