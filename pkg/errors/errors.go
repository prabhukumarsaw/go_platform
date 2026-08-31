package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// Standard Domain Error Codes
const (
	CodeNotFound       = "NOT_FOUND"
	CodeUnauthorized   = "UNAUTHORIZED"
	CodeForbidden      = "FORBIDDEN"
	CodeBadRequest     = "BAD_REQUEST"
	CodeConflict       = "CONFLICT"
	CodeInternal       = "INTERNAL_SERVER_ERROR"
	CodeValidation     = "VALIDATION_FAILED"
	CodeRateLimited    = "RATE_LIMIT_EXCEEDED"
	CodeTenantMismatch = "TENANT_ACCESS_DENIED"
	CodeMFAEnforced    = "MFA_VERIFICATION_REQUIRED"
)

// AppError represents a structured, domain-level application error.
type AppError struct {
	StatusCode int                    `json:"status_code"`
	Code       string                 `json:"code"`
	Message    string                 `json:"message"`
	Err        error                  `json:"-"`
	Details    map[string]interface{} `json:"details,omitempty"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// ─── Error Constructors ─────────────────────────

func New(statusCode int, code, message string, err error) *AppError {
	return &AppError{
		StatusCode: statusCode,
		Code:       code,
		Message:    message,
		Err:        err,
	}
}

func NotFound(resource string, err error) *AppError {
	return &AppError{
		StatusCode: http.StatusNotFound,
		Code:       CodeNotFound,
		Message:    fmt.Sprintf("%s not found", resource),
		Err:        err,
	}
}

func Unauthorized(message string, err error) *AppError {
	if message == "" {
		message = "Authentication required or credentials invalid"
	}
	return &AppError{
		StatusCode: http.StatusUnauthorized,
		Code:       CodeUnauthorized,
		Message:    message,
		Err:        err,
	}
}

func Forbidden(message string, err error) *AppError {
	if message == "" {
		message = "You do not have permission to access this resource"
	}
	return &AppError{
		StatusCode: http.StatusForbidden,
		Code:       CodeForbidden,
		Message:    message,
		Err:        err,
	}
}

func BadRequest(message string, err error) *AppError {
	return &AppError{
		StatusCode: http.StatusBadRequest,
		Code:       CodeBadRequest,
		Message:    message,
		Err:        err,
	}
}

func Conflict(message string, err error) *AppError {
	return &AppError{
		StatusCode: http.StatusConflict,
		Code:       CodeConflict,
		Message:    message,
		Err:        err,
	}
}

func Internal(message string, err error) *AppError {
	if message == "" {
		message = "An unexpected internal server error occurred"
	}
	return &AppError{
		StatusCode: http.StatusInternalServerError,
		Code:       CodeInternal,
		Message:    message,
		Err:        err,
	}
}

// As checks if an error is an *AppError.
func As(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}
