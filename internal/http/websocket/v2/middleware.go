package websocket

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

func CheckOrigin(origins ...string) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, origin := range origins {
				if err := checkOrigin(r, origin); err != nil {
					http.Error(w, err.Error(), http.StatusForbidden)
					return
				}
			}

			h.ServeHTTP(w, r)
		})
	}
}

func checkOrigin(r *http.Request, origins string) error {
	ohd := r.Header["Origin"]
	if len(ohd) == 0 {
		return errors.New("websocket: origin header not found")
	}

	u, err := url.Parse(ohd[0])
	if err != nil {
		return err
	}

	if !strings.EqualFold(u.Host, origins) {
		return fmt.Errorf("websocket: origin %s not allowed", u.Host)
	}

	return nil
}
