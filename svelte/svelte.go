package svelte

import (
	"embed"
	"io/fs"
	"net/http"
)

// TODO -- embed files from dist/assets
var (
	//go:embed all:dist/*.js
	fsys embed.FS
	
	ss  fs.FS
)

func init() {
	var err error
	if ss, err = fs.Sub(fsys, "./dist/assets/"); err != nil {
		panic(err)
	}
}

type Svelte struct {
	Prefix string
}

func (s *Svelte) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.StripPrefix(s.Prefix, http.FileServer(http.FS(ss))).ServeHTTP(w, r)
}