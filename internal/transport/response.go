package transport

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/domain"
)

type successEnvelope struct {
	Data any `json:"data"`
}
type errorEnvelope struct {
	Error apiError `json:"error"`
}
type apiError struct {
	Code    domain.ErrorCode `json:"code"`
	Message string           `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeData(w http.ResponseWriter, status int, data any) {
	writeJSON(w, status, successEnvelope{Data: data})
}

func writeError(w http.ResponseWriter, err error) {
	code := domain.ErrorCodeOf(err)
	status := http.StatusInternalServerError
	switch code {
	case domain.CodeInvalid:
		status = http.StatusBadRequest
	case domain.CodeNotFound:
		status = http.StatusNotFound
	case domain.CodeForbidden:
		status = http.StatusForbidden
	case domain.CodeConflict, domain.CodeInvalidState:
		status = http.StatusConflict
	}
	message := "服务内部错误"
	var business *domain.BusinessError
	if errors.As(err, &business) {
		message = business.Message
	}
	writeJSON(w, status, errorEnvelope{Error: apiError{Code: code, Message: message}})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return domain.NewError(domain.CodeInvalid, "请求 JSON 无效：%v", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return domain.NewError(domain.CodeInvalid, "请求只能包含一个 JSON 对象")
	}
	return nil
}

func requireMethod(w http.ResponseWriter, r *http.Request, methods ...string) bool {
	for _, method := range methods {
		if r.Method == method {
			return true
		}
	}
	w.Header().Set("Allow", methods[0])
	writeError(w, domain.NewError(domain.CodeInvalid, "不支持 HTTP 方法 %s", r.Method))
	return false
}
