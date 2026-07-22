// Package httpx holds small helpers for writing JSON responses and errors
// consistently across every handler.
package httpx

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
)

// ClientIP extracts the best-effort client IP (no port) from a request,
// honoring X-Forwarded-For. Returns "" when it cannot be determined, which
// callers can store as a NULL inet.
func ClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		first := xff
		if i := strings.IndexByte(xff, ','); i >= 0 {
			first = xff[:i]
		}
		if ip := net.ParseIP(strings.TrimSpace(first)); ip != nil {
			return ip.String()
		}
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		if ip := net.ParseIP(host); ip != nil {
			return ip.String()
		}
		return host
	}
	if ip := net.ParseIP(r.RemoteAddr); ip != nil {
		return ip.String()
	}
	return ""
}

// APIError is the uniform error envelope returned to the frontend.
type APIError struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *APIError) Error() string { return e.Message }

// NewError builds an APIError.
func NewError(status int, code, message string) *APIError {
	return &APIError{Status: status, Code: code, Message: message}
}

// Common error constructors.
func BadRequest(msg string) *APIError { return NewError(http.StatusBadRequest, "bad_request", msg) }
func Unauthorized(msg string) *APIError {
	return NewError(http.StatusUnauthorized, "unauthorized", msg)
}
func Forbidden(msg string) *APIError { return NewError(http.StatusForbidden, "forbidden", msg) }
func NotFound(msg string) *APIError  { return NewError(http.StatusNotFound, "not_found", msg) }
func Conflict(msg string) *APIError  { return NewError(http.StatusConflict, "conflict", msg) }
func Internal() *APIError {
	return NewError(http.StatusInternalServerError, "internal_error", "Something went wrong.")
}

// JSON writes v as a JSON response with the given status code.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("httpx: encode response: %v", err)
	}
}

// Error writes an error response. Non-APIError values are masked as a 500 so
// internal details never leak to clients.
func Error(w http.ResponseWriter, err error) {
	apiErr, ok := err.(*APIError)
	if !ok {
		log.Printf("httpx: unhandled error: %v", err)
		apiErr = Internal()
	}
	JSON(w, apiErr.Status, map[string]any{"error": apiErr})
}

// Decode reads a JSON request body into dst, rejecting unknown fields and
// oversized payloads.
func Decode(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return BadRequest("Invalid request body: " + err.Error())
	}
	return nil
}
