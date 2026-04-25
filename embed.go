package slotart

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed website/dist
var EmbeddedFS embed.FS

func StaticHandler() http.Handler {
	staticFS, _ := fs.Sub(EmbeddedFS, "website/dist")
	fileServer := http.FileServer(http.FS(staticFS))
	hashedAsset := []string{".css", ".js"}
	indexData, _ := fs.ReadFile(EmbeddedFS, "website/dist/index.html")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clean := strings.TrimPrefix(r.URL.Path, "/")
		if clean == "" || clean == "/" {
			clean = "index.html"
		}

		f, err := staticFS.Open(clean)
		if err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(indexData)
			return
		}
		f.Close()

		for _, ext := range hashedAsset {
			if path.Ext(clean) == ext {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				break
			}
		}
		fileServer.ServeHTTP(w, r)
	})
}
