package http

import "net/http"

type MiddleWare func(http.HandlerFunc) http.HandlerFunc

func Chain(hf http.HandlerFunc, mw ...MiddleWare) http.HandlerFunc {
	for _, m := range mw {
		hf = m(hf)
	}

	return hf
}
