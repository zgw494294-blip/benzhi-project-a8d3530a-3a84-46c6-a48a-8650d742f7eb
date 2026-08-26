package cross_operation_idempotency_cache_test

import (
	"bytes"
	"encoding/json"
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

func TestCrossOperationIdempotencyKeyCannotBypassMonitoring(t *testing.T) {
	store, err := repository.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	service := application.NewService(store, rules.NewEngine())
	handler := transport.NewServer(service, slog.New(slog.NewTextHandler(io.Discard, nil))).Handler()

	create := request(t, handler, http.MethodPost, "/api/cases", map[string]any{
		"expectedVersion": 0,
		"idempotencyKey":  "shared-command-key",
		"actor":           "监测员甲",
		"role":            "monitor",
		"name":            "幂等缓存边界测试",
		"siteCode":        "CACHE-01",
		"habitatType":     "盐沼",
		"baseline": []map[string]any{{
			"indicator": "植被覆盖率", "minimum": 60, "maximum": 100, "unit": "%",
		}},
	})
	if create.Code != http.StatusCreated {
		t.Fatalf("建档失败: status=%d body=%s", create.Code, create.Body.String())
	}

	var created struct {
		Data struct {
			Case struct {
				ID      string `json:"id"`
				Version int64  `json:"version"`
			} `json:"case"`
		} `json:"data"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	monitoring := request(t, handler, http.MethodPost, "/api/cases/"+created.Data.Case.ID+"/monitoring", map[string]any{
		"expectedVersion": created.Data.Case.Version,
		"idempotencyKey":  "shared-command-key",
		"actor":           "监测员甲",
		"role":            "monitor",
		"indicator":       "植被覆盖率",
		"observedValue":   80,
		"unit":            "%",
		"evidenceNote":    "样方证据 CACHE-EVIDENCE-01",
		"capturedBy":      "监测员甲",
	})
	if monitoring.Code != http.StatusConflict {
		t.Fatalf("跨操作复用 idempotencyKey 应返回 409，实际 status=%d body=%s", monitoring.Code, monitoring.Body.String())
	}
}

func request(t *testing.T, handler http.Handler, method, path string, payload any) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	return response
}
