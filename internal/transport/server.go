package transport

import (
	"log/slog"
	"net/http"
	"time"

	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/application"
)

type Server struct {
	app    *application.Service
	logger *slog.Logger
	mux    *http.ServeMux
}

func NewServer(app *application.Service, logger *slog.Logger) *Server {
	server := &Server{app: app, logger: logger, mux: http.NewServeMux()}
	server.routes()
	return server
}

func (s *Server) routes() {
	s.mux.HandleFunc("/", s.HandleIndex)
	s.mux.HandleFunc("/assets/app.css", s.HandleCSS)
	s.mux.HandleFunc("/assets/app.js", s.HandleJS)
	s.mux.HandleFunc("/healthz", s.HandleHealth)
	s.mux.HandleFunc("/api/cases", s.HandleCases)
	s.mux.HandleFunc("/api/cases/", s.HandleCaseRoutes)
	s.mux.HandleFunc("/api/certificates/", s.HandleCertificateAPI)
	s.mux.HandleFunc("/certificates/", s.HandleCertificatePage)
}

func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		s.mux.ServeHTTP(w, r)
		s.logger.Debug("HTTP 请求", "method", r.Method, "path", r.URL.Path, "duration", time.Since(started))
	})
}

func NewHTTPServer(address string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr: address, Handler: handler, ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout: 15 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second,
	}
}
