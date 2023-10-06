package svelte

import (
	"embed"
	"net/http"
)

var (
	//go:embed dist/bundle.js
	fsys embed.FS

	bundle []byte
)

func init() {
	var err error
	if bundle, err = fsys.ReadFile("dist/bundle.js"); err != nil {
		panic(err)
	}
}

func SvelteHandler(w http.ResponseWriter, r *http.Request) {
	// NOTE -- see if ETag will cache this
	w.Header().Set("Content-Type", "application/javascript")
	w.Write(bundle)
}
