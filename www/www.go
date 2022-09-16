package www

import (
	"net/http"

	h "github.com/hyphengolang/prelude/http"
)

type contextKey string

var (
	roomKey    = contextKey("rmx-fetch-pool")
	upgradeKey = contextKey("rmx-upgrade-http")
)

func chain(hf http.HandlerFunc, mw ...h.MiddleWare) http.HandlerFunc { return h.Chain(hf, mw...) }
