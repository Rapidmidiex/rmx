package api

import (
	"encoding/json"
	"net/http"
	"strings"
)

func isEmptyString(s string) bool {
	return len(strings.TrimSpace(s)) == 0
}

func parse(r *http.Request, out interface{}) error {
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&out)
	if err != nil {
		return err
	}

	return nil
}

type errorResponse struct {
	status  int
	message string
}

// custom error type for detecting internal application errors
func (e *errorResponse) Error() string {
	return e.message
}
