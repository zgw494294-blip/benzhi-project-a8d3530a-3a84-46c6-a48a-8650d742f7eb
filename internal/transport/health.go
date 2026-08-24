package transport

import "net/http"

func (s *Server) HandleHealth(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	writeData(w, http.StatusOK, map[string]any{"status": "ok", "ledger": s.app.Integrity()})
}
