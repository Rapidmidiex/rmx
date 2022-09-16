package www

import (
	"net/http"

	h "github.com/hyphengolang/prelude/http"
)

type contextKey string

// func (c *contextKey) String() string { return "context value " + c.string }

var (
	roomKey    = contextKey("ws-pool")
	upgradeKey = contextKey("http-upgrade")
)

func chain(hf http.HandlerFunc, mw ...h.MiddleWare) http.HandlerFunc { return h.Chain(hf, mw...) }
