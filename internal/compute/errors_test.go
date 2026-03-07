package compute

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestComputeError_Error(t *testing.T) {
	// Without wrapped error.
	err := &ComputeError{Code: ErrCodeNotFound, Message: "instance not found"}
	got := err.Error()
	if got != "NOT_FOUND: instance not found" {
		t.Errorf("Error() = %q, want %q", got, "NOT_FOUND: instance not found")
	}

	// With wrapped error.
	inner := fmt.Errorf("db connection failed")
	err2 := &ComputeError{Code: ErrCodeInternal, Message: "internal server error", Err: inner}
	got2 := err2.Error()
	if got2 != "INTERNAL_ERROR: internal server error: db connection failed" {
		t.Errorf("Error() = %q", got2)
	}
}

func TestComputeError_Unwrap(t *testing.T) {
	inner := fmt.Errorf("root cause")
	err := &ComputeError{Code: ErrCodeInternal, Message: "wrap", Err: inner}
	if !errors.Is(err, inner) {
		t.Error("Unwrap should return the inner error")
	}

	// Nil unwrap.
	err2 := &ComputeError{Code: ErrCodeNotFound, Message: "no wrap"}
	if err2.Unwrap() != nil {
		t.Error("Unwrap should return nil when no inner error")
	}
}

func TestNewInvalidRequestError(t *testing.T) {
	inner := fmt.Errorf("bad field")
	err := NewInvalidRequestError("invalid name", inner)
	if err.Code != ErrCodeInvalidRequest {
		t.Errorf("Code = %s, want %s", err.Code, ErrCodeInvalidRequest)
	}
	if err.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, want %d", err.StatusCode, http.StatusBadRequest)
	}
	if err.Err != inner {
		t.Error("Should wrap the inner error")
	}
}

func TestNewNotFoundError(t *testing.T) {
	err := NewNotFoundError("instance", 42)
	if err.Code != ErrCodeNotFound {
		t.Errorf("Code = %s, want %s", err.Code, ErrCodeNotFound)
	}
	if err.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode = %d, want %d", err.StatusCode, http.StatusNotFound)
	}
	if err.Message != "instance not found: 42" {
		t.Errorf("Message = %q", err.Message)
	}
}

func TestNewConflictError(t *testing.T) {
	err := NewConflictError("already exists")
	if err.Code != ErrCodeConflict {
		t.Errorf("Code = %s", err.Code)
	}
	if err.StatusCode != http.StatusConflict {
		t.Errorf("StatusCode = %d", err.StatusCode)
	}
}

func TestNewQuotaExceededError(t *testing.T) {
	err := NewQuotaExceededError("vcpus", 100)
	if err.Code != ErrCodeQuotaExceeded {
		t.Errorf("Code = %s", err.Code)
	}
	if err.StatusCode != http.StatusForbidden {
		t.Errorf("StatusCode = %d", err.StatusCode)
	}
	if err.Message != "quota exceeded for vcpus (limit: 100)" {
		t.Errorf("Message = %q", err.Message)
	}
}

func TestNewInsufficientResourcesError(t *testing.T) {
	err := NewInsufficientResourcesError("memory")
	if err.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("StatusCode = %d", err.StatusCode)
	}
}

func TestNewOperationFailedError(t *testing.T) {
	inner := fmt.Errorf("disk full")
	err := NewOperationFailedError("snapshot", inner)
	if err.Code != ErrCodeOperationFailed {
		t.Errorf("Code = %s", err.Code)
	}
	if err.Err != inner {
		t.Error("Should wrap the inner error")
	}
}

func TestNewTimeoutError(t *testing.T) {
	err := NewTimeoutError("live migration")
	if err.StatusCode != http.StatusGatewayTimeout {
		t.Errorf("StatusCode = %d", err.StatusCode)
	}
}

func TestNewInternalError(t *testing.T) {
	inner := fmt.Errorf("panic")
	err := NewInternalError(inner)
	if err.StatusCode != http.StatusInternalServerError {
		t.Errorf("StatusCode = %d", err.StatusCode)
	}
}

func TestIsNotFoundError(t *testing.T) {
	err := NewNotFoundError("volume", "v-1")
	if !IsNotFoundError(err) {
		t.Error("Should be a not found error")
	}
	if IsNotFoundError(fmt.Errorf("plain error")) {
		t.Error("Plain error should not be not found")
	}
	if IsNotFoundError(NewConflictError("conflict")) {
		t.Error("Conflict should not be not found")
	}
}

func TestIsQuotaExceededError(t *testing.T) {
	err := NewQuotaExceededError("instances", 10)
	if !IsQuotaExceededError(err) {
		t.Error("Should be quota exceeded")
	}
	if IsQuotaExceededError(NewNotFoundError("x", 1)) {
		t.Error("Not found should not be quota exceeded")
	}
}

func TestIsConflictError(t *testing.T) {
	err := NewConflictError("dup")
	if !IsConflictError(err) {
		t.Error("Should be conflict")
	}
}

func TestToErrorResponse(t *testing.T) {
	// ComputeError.
	err := NewNotFoundError("instance", "i-1")
	resp := ToErrorResponse(err)
	if resp.Error.Code != ErrCodeNotFound {
		t.Errorf("Code = %s", resp.Error.Code)
	}

	// Plain error.
	resp2 := ToErrorResponse(fmt.Errorf("unknown"))
	if resp2.Error.Code != ErrCodeInternal {
		t.Errorf("Code = %s, want %s", resp2.Error.Code, ErrCodeInternal)
	}
	if resp2.Error.Message != "an unexpected error occurred" {
		t.Errorf("Message = %q", resp2.Error.Message)
	}
}

func TestGetStatusCode(t *testing.T) {
	if got := GetStatusCode(NewNotFoundError("x", 1)); got != 404 {
		t.Errorf("got %d, want 404", got)
	}
	if got := GetStatusCode(NewConflictError("dup")); got != 409 {
		t.Errorf("got %d, want 409", got)
	}
	// Plain error defaults to 500.
	if got := GetStatusCode(fmt.Errorf("oops")); got != 500 {
		t.Errorf("got %d, want 500", got)
	}
}
