package wrapped_case_error_code_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/application"
	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/repository"
	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/rules"
	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/transport"
)

func TestWrappedCaseNotFoundKeepsHTTPErrorCode(t *testing.T) {
	store, err := repository.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := transport.NewServer(application.NewService(store, rules.NewEngine()), logger)

	tests := []struct {
		path string
		want int
	}{
		{path: "/api/cases/missing-case", want: http.StatusNotFound},
		{path: "/api/certificates/MISSING", want: http.StatusNotFound},
	}
	for _, tc := range tests {
		request := httptest.NewRequest(http.MethodGet, tc.path, nil)
		recorder := httptest.NewRecorder()
		server.Handler().ServeHTTP(recorder, request)
		if recorder.Code != tc.want {
			t.Fatalf("expected %s to return %d, got %d: %s", tc.path, tc.want, recorder.Code, recorder.Body.String())
		}
	}
}
