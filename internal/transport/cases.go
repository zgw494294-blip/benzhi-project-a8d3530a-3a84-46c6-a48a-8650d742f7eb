package transport

import (
	"net/http"

	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/application"
	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/domain"
)

type requestMeta struct {
	ExpectedVersion int64       `json:"expectedVersion"`
	IdempotencyKey  string      `json:"idempotencyKey"`
	Actor           string      `json:"actor"`
	Role            domain.Role `json:"role"`
}

func (m requestMeta) application() application.CommandMeta {
	return application.CommandMeta{ExpectedVersion: m.ExpectedVersion, IdempotencyKey: m.IdempotencyKey, Actor: m.Actor, Role: m.Role}
}

type createCaseRequest struct {
	requestMeta
	Name        string                  `json:"name"`
	SiteCode    string                  `json:"siteCode"`
	HabitatType string                  `json:"habitatType"`
	Baseline    []domain.IndicatorRange `json:"baseline"`
}

func (s *Server) HandleCases(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeData(w, http.StatusOK, s.app.ListCases())
	case http.MethodPost:
		var request createCaseRequest
		if err := decodeJSON(w, r, &request); err != nil {
			writeError(w, err)
			return
		}
		result, err := s.app.CreateCase(application.CreateCaseCommand{
			Meta: request.application(), Name: request.Name, SiteCode: request.SiteCode,
			HabitatType: request.HabitatType, Baseline: request.Baseline,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		status := http.StatusCreated
		if result.Idempotent {
			status = http.StatusOK
		}
		writeData(w, status, result)
	default:
		requireMethod(w, r, http.MethodGet, http.MethodPost)
	}
}
