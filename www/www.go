package www

import (
	"net/http"

	h "github.com/hyphengolang/prelude/http"

	"github.com/rog-golang-buddies/rapidmidiex/internal/suid"
)

type contextKey string

var (
	roomKey    = contextKey("rmx-fetch-pool")
	upgradeKey = contextKey("rmx-upgrade-http")
)

func chain(hf http.HandlerFunc, mw ...h.MiddleWare) http.HandlerFunc { return h.Chain(hf, mw...) }

type Session struct {
	ID    suid.SUID   `json:"id"`
	Name  string      `json:"name,omitempty"`
	Users []suid.SUID `json:"users,omitempty"`
}

type User struct {
	ID   suid.SUID `json:"id"`
	Name string    `json:"name,omitempty"`
	/* More fields can belong here */
}
