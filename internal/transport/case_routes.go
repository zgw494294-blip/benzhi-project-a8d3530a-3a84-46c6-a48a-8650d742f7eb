package transport

import (
	"net/http"
	"strings"
	"time"

	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/application"
	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/domain"
)

type monitoringRequest struct {
	requestMeta
	Indicator        string    `json:"indicator"`
	ObservedValue    float64   `json:"observedValue"`
	Unit             string    `json:"unit"`
	EvidenceNote     string    `json:"evidenceNote"`
	CapturedBy       string    `json:"capturedBy"`
	CapturedAt       time.Time `json:"capturedAt"`
	RemediationOwner string    `json:"remediationOwner"`
	RemediationDueAt time.Time `json:"remediationDueAt"`
}

type retestRequest struct {
	requestMeta
	Owner         string  `json:"owner"`
	ObservedValue float64 `json:"observedValue"`
	EvidenceNote  string  `json:"evidenceNote"`
}

type acceptanceRequest struct {
	requestMeta
	Reviewer   string                    `json:"reviewer"`
	Decision   domain.AcceptanceDecision `json:"decision"`
	ReviewNote string                    `json:"reviewNote"`
}

func (s *Server) HandleCaseRoutes(w http.ResponseWriter, r *http.Request) {
	parts := splitPath(strings.TrimPrefix(r.URL.Path, "/api/cases/"))
	if len(parts) == 0 {
		http.NotFound(w, r)
		return
	}
	caseID := parts[0]
	if len(parts) == 1 {
		if !requireMethod(w, r, http.MethodGet) {
			return
		}
		detail, err := s.app.GetCase(caseID)
		if err != nil {
			writeError(w, err)
			return
		}
		writeData(w, http.StatusOK, detail)
		return
	}
	if len(parts) == 2 && parts[1] == "monitoring" {
		s.handleMonitoring(w, r, caseID)
		return
	}
	if len(parts) == 2 && parts[1] == "acceptance" {
		s.handleAcceptance(w, r, caseID)
		return
	}
	if len(parts) == 4 && parts[1] == "remediations" && parts[3] == "retest" {
		s.handleRetest(w, r, caseID, parts[2])
		return
	}
	http.NotFound(w, r)
}

func splitPath(path string) []string {
	raw := strings.Split(strings.Trim(path, "/"), "/")
	result := make([]string, 0, len(raw))
	for _, part := range raw {
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func (s *Server) handleMonitoring(w http.ResponseWriter, r *http.Request, caseID string) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var request monitoringRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, err)
		return
	}
	result, err := s.app.SubmitMonitoring(application.SubmitMonitoringCommand{
		Meta: request.application(), CaseID: caseID, Indicator: request.Indicator, ObservedValue: request.ObservedValue,
		Unit: request.Unit, EvidenceNote: request.EvidenceNote, CapturedBy: request.CapturedBy, CapturedAt: request.CapturedAt,
		RemediationOwner: request.RemediationOwner, RemediationDueAt: request.RemediationDueAt,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusCreated, result)
}

func (s *Server) handleRetest(w http.ResponseWriter, r *http.Request, caseID, actionID string) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var request retestRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, err)
		return
	}
	result, err := s.app.SubmitRetest(application.SubmitRetestCommand{
		Meta: request.application(), CaseID: caseID, ActionID: actionID, Owner: request.Owner,
		ObservedValue: request.ObservedValue, EvidenceNote: request.EvidenceNote,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusCreated, result)
}

func (s *Server) handleAcceptance(w http.ResponseWriter, r *http.Request, caseID string) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var request acceptanceRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, err)
		return
	}
	result, err := s.app.SubmitAcceptance(application.SubmitAcceptanceCommand{
		Meta: request.application(), CaseID: caseID, Reviewer: request.Reviewer,
		Decision: request.Decision, ReviewNote: request.ReviewNote,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusCreated, result)
}
