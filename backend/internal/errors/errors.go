package errors

import (
	"encoding/json"
	"net/http"
)

// ErrorCode represents a standardized error code
type ErrorCode string

const (
	CodeBadRequest       ErrorCode = "BAD_REQUEST"
	CodeUnauthorized     ErrorCode = "UNAUTHORIZED"
	CodeForbidden        ErrorCode = "FORBIDDEN"
	CodeNotFound         ErrorCode = "NOT_FOUND"
	CodeConflict         ErrorCode = "CONFLICT"
	CodeValidation       ErrorCode = "VALIDATION_ERROR"
	CodeInternal         ErrorCode = "INTERNAL_ERROR"
	CodeInsufficientStock ErrorCode = "INSUFFICIENT_STOCK"
	CodeInvalidStatus    ErrorCode = "INVALID_STATUS"
)

// APIError represents a standardized API error response
type APIError struct {
	Code    ErrorCode         `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

func (e *APIError) Error() string {
	return e.Message
}

// New creates a new API error
func New(code ErrorCode, message string) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
	}
}

// WithDetails adds details to the error
func (e *APIError) WithDetails(details map[string]string) *APIError {
	e.Details = details
	return e
}

// Common errors
var (
	ErrNotFound     = New(CodeNotFound, "resource not found")
	ErrUnauthorized = New(CodeUnauthorized, "unauthorized")
	ErrForbidden    = New(CodeForbidden, "forbidden")
	ErrInternal     = New(CodeInternal, "internal server error")
)

// WriteJSON writes an error response as JSON
func WriteJSON(w http.ResponseWriter, statusCode int, err *APIError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(err)
}

// BadRequest writes a 400 error response
func BadRequest(w http.ResponseWriter, message string) {
	WriteJSON(w, http.StatusBadRequest, New(CodeBadRequest, message))
}

// Unauthorized writes a 401 error response
func Unauthorized(w http.ResponseWriter, message string) {
	WriteJSON(w, http.StatusUnauthorized, New(CodeUnauthorized, message))
}

// Forbidden writes a 403 error response
func Forbidden(w http.ResponseWriter, message string) {
	WriteJSON(w, http.StatusForbidden, New(CodeForbidden, message))
}

// NotFound writes a 404 error response
func NotFound(w http.ResponseWriter, message string) {
	WriteJSON(w, http.StatusNotFound, New(CodeNotFound, message))
}

// Conflict writes a 409 error response
func Conflict(w http.ResponseWriter, message string) {
	WriteJSON(w, http.StatusConflict, New(CodeConflict, message))
}

// Validation writes a 422 error response with field details
func Validation(w http.ResponseWriter, message string, details map[string]string) {
	WriteJSON(w, http.StatusUnprocessableEntity, New(CodeValidation, message).WithDetails(details))
}

// Internal writes a 500 error response
func Internal(w http.ResponseWriter) {
	WriteJSON(w, http.StatusInternalServerError, ErrInternal)
}
