package transport

import (
	"embed"
	"net/http"
)

//go:embed web/index.html web/app.css web/app.js
var webAssets embed.FS

func (s *Server) HandleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	serveEmbedded(w, r, "web/index.html", "text/html; charset=utf-8")
}

func (s *Server) HandleCSS(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	serveEmbedded(w, r, "web/app.css", "text/css; charset=utf-8")
}

func (s *Server) HandleJS(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	serveEmbedded(w, r, "web/app.js", "text/javascript; charset=utf-8")
}

func serveEmbedded(w http.ResponseWriter, r *http.Request, name, contentType string) {
	content, err := webAssets.ReadFile(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(content)
}
