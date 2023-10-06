package static

import (
	"embed"
	"io/fs"
	"net/http"
)

var (
	//go:embed all:*
	fsys embed.FS
	
	ss  fs.FS
)

func init() {
	var err error
	if ss, err = fs.Sub(fsys, "."); err != nil {
		panic(err)
	}
}

type Static struct {
	Prefix string
}

func (s *Static) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.StripPrefix(s.Prefix, http.FileServer(http.FS(ss))).ServeHTTP(w, r)
}