package webui

import (
	"bytes"
	"embed"
	"io/fs"
	"net/http"
	"path"
	"runtime"
	"strings"
)

//go:embed dist
var assets embed.FS

// FS exposes the shared frontend build to the service and desktop host.
func FS() fs.FS {
	root, err := fs.Sub(assets, "dist")
	if err != nil {
		panic(err)
	}
	return root
}

func runtimeEdition() string {
	if runtime.GOOS == "linux" {
		return "fnos"
	}
	return "mac"
}

func applyEditionBrand(index []byte, edition string) []byte {
	if edition != "fnos" {
		return index
	}
	index = bytes.ReplaceAll(index, []byte("<title>OpenSurge for Mac</title>"), []byte("<title>OpenSurge for fnOS</title>"))
	index = bytes.ReplaceAll(index, []byte(`name="opensurge-edition" content="mac"`), []byte(`name="opensurge-edition" content="fnos"`))
	index = bytes.ReplaceAll(index, []byte("/opensurge-icon.png"), []byte("/opensurge-fnos-icon.png"))
	return index
}

func Handler() http.Handler {
	root := FS()
	files := http.FileServer(http.FS(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if requested != "." && requested != "" {
			if _, err := fs.Stat(root, requested); err == nil {
				files.ServeHTTP(w, r)
				return
			}
		}
		index, err := fs.ReadFile(root, "index.html")
		if err != nil {
			http.Error(w, "Web UI is unavailable", http.StatusServiceUnavailable)
			return
		}
		index = applyEditionBrand(index, runtimeEdition())
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(index)
	})
}
