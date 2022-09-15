// * most of this isn't required as I have already got a function that handles http responses
// * file:jam_tmp.go is dependant therefore cannot delete this straight away
package tmp

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
