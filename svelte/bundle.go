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

// BundleHandler will handle sending the svelte build to the client.
func BundleHandler(w http.ResponseWriter, r *http.Request) {
	// e := "checksum"
	// w.Header().Set("ETag", e)
	// w.Header().Set("Cache-Control", "max-age=2592000") // 30d

	// if match := r.Header.Get("If-None-Match"); match != "" {
	// 	if strings.Contains(match, e) {
	// 		w.WriteHeader(http.StatusNotModified)
	// 		return
	// 	}
	// }

	// NOTE -- see if ETag will cache this
	w.Header().Set("Content-Type", "application/javascript")
	w.Write(bundle)
}
