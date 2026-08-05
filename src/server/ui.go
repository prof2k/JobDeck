package main

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed webdist
var embeddedUI embed.FS

func embeddedUIHandler() (http.Handler, error) {
	dist, err := fs.Sub(embeddedUI, "webdist")
	if err != nil {
		return nil, fmt.Errorf("open embedded UI: %w", err)
	}
	files := http.FileServer(http.FS(dist))
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestedPath := strings.TrimPrefix(path.Clean(request.URL.Path), "/")
		if requestedPath == "." || requestedPath == "" {
			requestedPath = "index.html"
		}
		if _, err := fs.Stat(dist, requestedPath); err == nil {
			if requestedPath == "sw.js" || requestedPath == "index.html" {
				writer.Header().Set("Cache-Control", "no-cache")
			}
			files.ServeHTTP(writer, request)
			return
		}

		request.URL.Path = "/"
		writer.Header().Set("Cache-Control", "no-cache")
		files.ServeHTTP(writer, request)
	}), nil
}
