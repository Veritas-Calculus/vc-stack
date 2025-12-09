// Package compute provides error types and handling.
package compute

import (
	"errors"
	"fmt"
	"net/http"
)

// Error codes for compute operations.
const (
	ErrCodeInvalidRequest        = "INVALID_REQUEST"
	ErrCodeNotFound              = "NOT_FOUND"
	ErrCodeConflict              = "CONFLICT"
	ErrCodeQuotaExceeded         = "QUOTA_EXCEEDED"
	ErrCodeInsufficientResources = "INSUFFICIENT_RESOURCES"
	ErrCodeOperationFailed       = "OPERATION_FAILED"
	ErrCodeTimeout               = "TIMEOUT"
	ErrCodeInternal              = "INTERNAL_ERROR"
)

// ComputeError represents a compute service error.
type ComputeError struct {
	Code       string
	Message    string
	StatusCode int
	Err        error
}

// Error implements the error interface.
func (e *ComputeError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the wrapped error.
func (e *ComputeError) Unwrap() error {
	return e.Err
}

// NewInvalidRequestError creates an invalid request error.
func NewInvalidRequestError(message string, err error) *ComputeError {
	return &ComputeError{
		Code:       ErrCodeInvalidRequest,
		Message:    message,
		StatusCode: http.StatusBadRequest,
		Err:        err,
	}
}

// NewNotFoundError creates a not found error.
func NewNotFoundError(resource string, id interface{}) *ComputeError {
	return &ComputeError{
		Code:       ErrCodeNotFound,
		Message:    fmt.Sprintf("%s not found: %v", resource, id),
		StatusCode: http.StatusNotFound,
	}
}

// NewConflictError creates a conflict error.
func NewConflictError(message string) *ComputeError {
	return &ComputeError{
		Code:       ErrCodeConflict,
		Message:    message,
		StatusCode: http.StatusConflict,
	}
}

// NewQuotaExceededError creates a quota exceeded error.
func NewQuotaExceededError(resource string, limit int) *ComputeError {
	return &ComputeError{
		Code:       ErrCodeQuotaExceeded,
		Message:    fmt.Sprintf("quota exceeded for %s (limit: %d)", resource, limit),
		StatusCode: http.StatusForbidden,
	}
}

// NewInsufficientResourcesError creates an insufficient resources error.
func NewInsufficientResourcesError(resource string) *ComputeError {
	return &ComputeError{
		Code:       ErrCodeInsufficientResources,
		Message:    fmt.Sprintf("insufficient %s available", resource),
		StatusCode: http.StatusServiceUnavailable,
	}
}

// NewOperationFailedError creates an operation failed error.
func NewOperationFailedError(operation string, err error) *ComputeError {
	return &ComputeError{
		Code:       ErrCodeOperationFailed,
		Message:    fmt.Sprintf("%s failed", operation),
		StatusCode: http.StatusInternalServerError,
		Err:        err,
	}
}

// NewTimeoutError creates a timeout error.
func NewTimeoutError(operation string) *ComputeError {
	return &ComputeError{
		Code:       ErrCodeTimeout,
		Message:    fmt.Sprintf("%s timed out", operation),
		StatusCode: http.StatusGatewayTimeout,
	}
}

// NewInternalError creates an internal error.
func NewInternalError(err error) *ComputeError {
	return &ComputeError{
		Code:       ErrCodeInternal,
		Message:    "internal server error",
		StatusCode: http.StatusInternalServerError,
		Err:        err,
	}
}

// IsNotFoundError checks if an error is a not found error.
func IsNotFoundError(err error) bool {
	var computeErr *ComputeError
	if errors.As(err, &computeErr) {
		return computeErr.Code == ErrCodeNotFound
	}
	return false
}

// IsQuotaExceededError checks if an error is a quota exceeded error.
func IsQuotaExceededError(err error) bool {
	var computeErr *ComputeError
	if errors.As(err, &computeErr) {
		return computeErr.Code == ErrCodeQuotaExceeded
	}
	return false
}

// IsConflictError checks if an error is a conflict error.
func IsConflictError(err error) bool {
	var computeErr *ComputeError
	if errors.As(err, &computeErr) {
		return computeErr.Code == ErrCodeConflict
	}
	return false
}

// ErrorResponse represents an error response body.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error details.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ToErrorResponse converts a ComputeError to ErrorResponse.
func ToErrorResponse(err error) ErrorResponse {
	var computeErr *ComputeError
	if errors.As(err, &computeErr) {
		return ErrorResponse{
			Error: ErrorDetail{
				Code:    computeErr.Code,
				Message: computeErr.Message,
			},
		}
	}

	return ErrorResponse{
		Error: ErrorDetail{
			Code:    ErrCodeInternal,
			Message: "an unexpected error occurred",
		},
	}
}

// GetStatusCode extracts HTTP status code from error.
func GetStatusCode(err error) int {
	var computeErr *ComputeError
	if errors.As(err, &computeErr) {
		return computeErr.StatusCode
	}
	return http.StatusInternalServerError
}
