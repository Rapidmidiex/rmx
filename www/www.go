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
	Name      string      `json:"name,omitempty"`
	SessionID suid.SUID   `json:"sessionId,omitempty"`
	Users     []suid.SUID `json:"users,omitempty"`
}
