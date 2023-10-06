// https://charly3pins.dev/blog/learn-how-to-use-the-embed-package-in-go-by-building-a-web-page-easily/
package html

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
	"text/template"
)

var (
	//go:embed all:partials
	fsys embed.FS

	pm map[string]*template.Template
)

func init() {
	if pm == nil {
		pm = make(map[string]*template.Template)
	}

	ff, err := fs.ReadDir(fsys, "partials")
	if err != nil {
		panic(err)
	}

	// TODO
	// 1. traverse nested folders [href](https://yourbasic.org/golang/list-files-in-directory/)
	for _, f := range ff {
		if f.IsDir() {
			continue
		}

		// 2. ignore files with _*.html pattern
		if strings.HasPrefix(f.Name(), "_") {
			continue
		}

		pt, err := template.ParseFS(fsys, "partials/"+f.Name(), "partials/_*.html")
		if err != nil {
			panic(err)
		}

		// call without extension
		filename := fileNameWithoutExtSliceNotation(f.Name())
		pm[filename] = pt
	}

	// fmt.Println(pm)
}

func Execute(wr io.Writer, name string, data any) error {
	t, ok := pm[name]
	if !ok {
		return fmt.Errorf("partial with name %s not found", name)
	}

	err := t.Execute(wr, data)
	if err != nil {
		return fmt.Errorf("error writing to output: %w", err)
	}

	return nil
}

// fileNameWithoutExtSliceNotation - https://freshman.tech/snippets/go/filename-no-extension/
func fileNameWithoutExtSliceNotation(filename string) string {
	return filename[:len(filename)-len(filepath.Ext(filename))]
}