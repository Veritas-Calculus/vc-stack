// Package apierrors provides standardized error codes and structured error responses
// for the VC Stack API. Follows AWS-style error code conventions with machine-readable
// codes, human-readable messages, and HTTP status mapping.
//
// Usage:
//
//	apierrors.Respond(c, apierrors.ErrNotFound("instance", id))
//	apierrors.Respond(c, apierrors.ErrQuotaExceeded("vcpus", 32, 24))
//	apierrors.Respond(c, apierrors.ErrValidation("name is required"))
package apierrors

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIError represents a structured error response.
// All API errors follow this format for consistency.
type APIError struct {
	// HTTP status code (not serialized in body, used for response).
	HTTPStatus int `json:"-"`
	// Machine-readable error code (e.g., "ResourceNotFound", "QuotaExceeded").
	Code string `json:"code"`
	// Human-readable error message.
	Message string `json:"message"`
	// Optional detail providing additional context.
	Detail string `json:"detail,omitempty"`
	// Optional field that caused the error (for validation errors).
	Field string `json:"field,omitempty"`
	// Optional request ID for tracing (populated by middleware).
	RequestID string `json:"request_id,omitempty"`

	// Backward compatibility: also include "error" key for existing clients.
	ErrorCompat string `json:"error"`
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Code, e.Message, e.Detail)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Respond sends the error as a JSON response and aborts the request.
func Respond(c *gin.Context, err *APIError) {
	// Populate request ID from context if available.
	if reqID, exists := c.Get("request_id"); exists {
		if id, ok := reqID.(string); ok {
			err.RequestID = id
		}
	}
	// Also check header.
	if err.RequestID == "" {
		err.RequestID = c.GetHeader("X-Request-ID")
	}

	c.JSON(err.HTTPStatus, err)
	c.Abort()
}

// RespondWithData sends an error with additional data fields.
func RespondWithData(c *gin.Context, err *APIError, data map[string]interface{}) {
	if reqID, exists := c.Get("request_id"); exists {
		if id, ok := reqID.(string); ok {
			err.RequestID = id
		}
	}

	resp := gin.H{
		"code":       err.Code,
		"message":    err.Message,
		"error":      err.ErrorCompat,
		"request_id": err.RequestID,
	}
	if err.Detail != "" {
		resp["detail"] = err.Detail
	}
	for k, v := range data {
		resp[k] = v
	}
	c.JSON(err.HTTPStatus, resp)
	c.Abort()
}

// --- Error Code Constants ---

// Authentication & Authorization errors (401, 403).
const (
	CodeAuthRequired       = "AuthenticationRequired"
	CodeInvalidCredentials = "InvalidCredentials"
	CodeTokenExpired       = "TokenExpired"
	CodeTokenInvalid       = "TokenInvalid"
	CodeAccessDenied       = "AccessDenied"
	CodeRateLimitExceeded  = "RateLimitExceeded"
)

// Validation errors (400).
const (
	CodeValidationFailed = "ValidationFailed"
	CodeInvalidParameter = "InvalidParameter"
	CodeMissingParameter = "MissingRequired"
	CodeInvalidFormat    = "InvalidFormat"
)

// Resource errors (404, 409).
const (
	CodeResourceNotFound  = "ResourceNotFound"
	CodeResourceExists    = "ResourceAlreadyExists"
	CodeResourceInUse     = "ResourceInUse"
	CodeResourceProtected = "ResourceProtected"
	CodeStateConflict     = "StateConflict"
)

// Quota and limit errors (403, 409, 429).
const (
	CodeQuotaExceeded = "QuotaExceeded"
	CodeLimitExceeded = "LimitExceeded"
)

// Operation errors (500, 502, 503).
const (
	CodeInternalError   = "InternalError"
	CodeServiceUnavail  = "ServiceUnavailable"
	CodeOperationFailed = "OperationFailed"
	CodeUpstreamError   = "UpstreamError"
	CodeDatabaseError   = "DatabaseError"
	CodeStorageError    = "StorageError"
)

// Compute-specific errors.
const (
	CodeInstanceNotFound     = "InstanceNotFound"
	CodeInstanceStateInvalid = "InvalidInstanceState"
	CodeFlavorNotFound       = "FlavorNotFound"
	CodeImageNotFound        = "ImageNotFound"
	CodeImageProtected       = "ImageProtected"
	CodeImageInUse           = "ImageInUse"
	CodeMigrationFailed      = "MigrationFailed"
	CodeHostNotFound         = "HostNotFound"
	CodeNoHostAvailable      = "NoHostAvailable"
)

// Network-specific errors.
const (
	CodeNetworkNotFound    = "NetworkNotFound"
	CodeSubnetNotFound     = "SubnetNotFound"
	CodePortNotFound       = "PortNotFound"
	CodeCIDRConflict       = "CIDRConflict"
	CodeIPExhausted        = "IPAddressExhausted"
	CodeSecGroupNotFound   = "SecurityGroupNotFound"
	CodeFloatingIPNotFound = "FloatingIPNotFound"
)

// Storage-specific errors.
const (
	CodeVolumeNotFound   = "VolumeNotFound"
	CodeVolumeInUse      = "VolumeInUse"
	CodeSnapshotNotFound = "SnapshotNotFound"
)

// --- Factory Functions ---

// New creates a custom APIError.
func New(httpStatus int, code, message string) *APIError {
	return &APIError{
		HTTPStatus:  httpStatus,
		Code:        code,
		Message:     message,
		ErrorCompat: message,
	}
}

// WithDetail adds detail to an error.
func (e *APIError) WithDetail(detail string) *APIError {
	e.Detail = detail
	return e
}

// WithField adds the field name to an error.
func (e *APIError) WithField(field string) *APIError {
	e.Field = field
	return e
}

// --- Common Error Constructors ---

// Authentication errors.

func ErrAuthRequired(msg string) *APIError {
	if msg == "" {
		msg = "authentication is required"
	}
	return New(http.StatusUnauthorized, CodeAuthRequired, msg)
}

func ErrInvalidCredentials() *APIError {
	return New(http.StatusUnauthorized, CodeInvalidCredentials, "invalid username or password")
}

func ErrTokenExpired() *APIError {
	return New(http.StatusUnauthorized, CodeTokenExpired, "token has expired")
}

func ErrTokenInvalid() *APIError {
	return New(http.StatusUnauthorized, CodeTokenInvalid, "invalid or malformed token")
}

func ErrAccessDenied(reason string) *APIError {
	msg := "access denied"
	e := New(http.StatusForbidden, CodeAccessDenied, msg)
	if reason != "" {
		e.Detail = reason
	}
	return e
}

func ErrRateLimited() *APIError {
	return New(http.StatusTooManyRequests, CodeRateLimitExceeded, "rate limit exceeded, try again later")
}

// Validation errors.

func ErrValidation(msg string) *APIError {
	return New(http.StatusBadRequest, CodeValidationFailed, msg)
}

func ErrInvalidParam(field, detail string) *APIError {
	e := New(http.StatusBadRequest, CodeInvalidParameter, fmt.Sprintf("invalid parameter: %s", field))
	e.Field = field
	e.Detail = detail
	return e
}

func ErrMissingParam(field string) *APIError {
	e := New(http.StatusBadRequest, CodeMissingParameter, fmt.Sprintf("missing required parameter: %s", field))
	e.Field = field
	return e
}

// Resource errors.

func ErrNotFound(resourceType, identifier string) *APIError {
	msg := fmt.Sprintf("%s not found", resourceType)
	code := CodeResourceNotFound

	// Use specific codes for common types.
	switch resourceType {
	case "instance":
		code = CodeInstanceNotFound
	case "flavor":
		code = CodeFlavorNotFound
	case "image":
		code = CodeImageNotFound
	case "host":
		code = CodeHostNotFound
	case "network":
		code = CodeNetworkNotFound
	case "subnet":
		code = CodeSubnetNotFound
	case "port":
		code = CodePortNotFound
	case "volume":
		code = CodeVolumeNotFound
	case "snapshot":
		code = CodeSnapshotNotFound
	case "security group":
		code = CodeSecGroupNotFound
	case "floating ip":
		code = CodeFloatingIPNotFound
	}

	e := New(http.StatusNotFound, code, msg)
	if identifier != "" {
		e.Detail = fmt.Sprintf("id: %s", identifier)
	}
	return e
}

func ErrAlreadyExists(resourceType, name string) *APIError {
	return New(http.StatusConflict, CodeResourceExists,
		fmt.Sprintf("%s with name %q already exists", resourceType, name))
}

func ErrResourceInUse(resourceType string, dependentCount int64) *APIError {
	e := New(http.StatusConflict, CodeResourceInUse,
		fmt.Sprintf("%s is in use and cannot be deleted", resourceType))
	if dependentCount > 0 {
		e.Detail = fmt.Sprintf("referenced by %d active resources", dependentCount)
	}
	return e
}

func ErrResourceProtected(resourceType string) *APIError {
	return New(http.StatusForbidden, CodeResourceProtected,
		fmt.Sprintf("%s is protected and cannot be modified or deleted", resourceType))
}

func ErrStateConflict(resourceType, currentState, requiredState string) *APIError {
	return New(http.StatusConflict, CodeStateConflict,
		fmt.Sprintf("%s is in state %q, requires %q", resourceType, currentState, requiredState))
}

// Quota errors.

func ErrQuotaExceeded(resource string, limit, requested int) *APIError {
	e := New(http.StatusForbidden, CodeQuotaExceeded,
		fmt.Sprintf("quota exceeded for %s", resource))
	e.Detail = fmt.Sprintf("limit: %d, requested: %d", limit, requested)
	return e
}

// Operation errors.

func ErrInternal(detail string) *APIError {
	e := New(http.StatusInternalServerError, CodeInternalError, "an internal error occurred")
	if detail != "" {
		e.Detail = detail
	}
	return e
}

func ErrDatabase(operation string) *APIError {
	return New(http.StatusInternalServerError, CodeDatabaseError,
		fmt.Sprintf("database error during %s", operation))
}

func ErrOperationFailed(operation, detail string) *APIError {
	e := New(http.StatusInternalServerError, CodeOperationFailed,
		fmt.Sprintf("operation failed: %s", operation))
	e.Detail = detail
	return e
}

func ErrServiceUnavailable(service string) *APIError {
	return New(http.StatusServiceUnavailable, CodeServiceUnavail,
		fmt.Sprintf("service %s is currently unavailable", service))
}

func ErrNoHostAvailable() *APIError {
	return New(http.StatusServiceUnavailable, CodeNoHostAvailable,
		"no compute host available for scheduling")
}
