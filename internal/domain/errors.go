package domain

import (
	"errors"
	"fmt"
)

type ErrorCode string

const (
	CodeInvalid      ErrorCode = "invalid_input"
	CodeConflict     ErrorCode = "version_conflict"
	CodeNotFound     ErrorCode = "not_found"
	CodeForbidden    ErrorCode = "forbidden"
	CodeInvalidState ErrorCode = "invalid_state"
	CodeIntegrity    ErrorCode = "integrity_error"
)

type BusinessError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

func (e *BusinessError) Error() string { return e.Message }

func NewError(code ErrorCode, format string, args ...any) error {
	return &BusinessError{Code: code, Message: fmt.Sprintf(format, args...)}
}

func ErrorCodeOf(err error) ErrorCode {
	var business *BusinessError
	if errors.As(err, &business) {
		return business.Code
	}
	return CodeIntegrity
}
