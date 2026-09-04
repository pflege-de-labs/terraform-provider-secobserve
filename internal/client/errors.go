package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

// APIError is a non-2xx response from SecObserve.
type APIError struct {
	Method string
	Path   string
	Status int

	// Detail is the top-level "detail" or "message" string DRF returns for
	// permission failures, 404s and the 409 conflicts raised by protected
	// foreign keys.
	Detail string

	// FieldErrors maps a serializer field name to its validation messages.
	// DRF returns these for 400 responses. The pseudo-field "non_field_errors"
	// carries cross-field validation failures.
	FieldErrors map[string][]string

	// RetryAfter is the parsed Retry-After header, set when the backend
	// throttled the request.
	RetryAfter time.Duration

	// Body is the raw response, kept for error messages we cannot classify.
	Body string
}

func (e *APIError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s: HTTP %d", e.Method, e.Path, e.Status)
	if e.Detail != "" {
		fmt.Fprintf(&b, ": %s", e.Detail)
	}
	if len(e.FieldErrors) > 0 {
		fields := make([]string, 0, len(e.FieldErrors))
		for field := range e.FieldErrors {
			fields = append(fields, field)
		}
		sort.Strings(fields)
		parts := make([]string, 0, len(fields))
		for _, field := range fields {
			parts = append(parts, field+": "+strings.Join(e.FieldErrors[field], "; "))
		}
		fmt.Fprintf(&b, ": %s", strings.Join(parts, ", "))
	}
	if e.Detail == "" && len(e.FieldErrors) == 0 && e.Body != "" {
		fmt.Fprintf(&b, ": %s", truncate(e.Body, 512))
	}
	return b.String()
}

// NotFound reports whether err is a 404. Read implementations use this to
// remove a resource from state instead of failing.
func NotFound(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound
}

// Conflict reports whether err is a 409, which SecObserve returns when a
// delete is blocked by a protected or restricted relation.
func Conflict(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.Status == http.StatusConflict
}

// Forbidden reports whether err is a 403, usually a missing permission on the
// provider's token rather than a bad configuration.
func Forbidden(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.Status == http.StatusForbidden
}

func newAPIError(method, path string, resp *http.Response) *APIError {
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	apiErr := &APIError{
		Method: method,
		Path:   path,
		Status: resp.StatusCode,
		Body:   string(raw),
	}

	if seconds, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil && seconds >= 0 {
		apiErr.RetryAfter = time.Duration(seconds) * time.Second
	}

	// DRF error bodies come in three shapes: {"detail": "..."},
	// {"message": "..."} (the SecObserve 409 handler) and
	// {"field": ["msg", ...]} for validation failures. A single body can mix
	// the last two, so parse into a generic map.
	var decoded map[string]json.RawMessage
	if json.Unmarshal(raw, &decoded) != nil {
		return apiErr
	}

	for key, value := range decoded {
		if key == "detail" || key == "message" {
			var detail string
			if json.Unmarshal(value, &detail) == nil {
				apiErr.Detail = detail
				continue
			}
		}
		if messages, ok := decodeMessages(value); ok {
			if apiErr.FieldErrors == nil {
				apiErr.FieldErrors = map[string][]string{}
			}
			apiErr.FieldErrors[key] = messages
		}
	}

	return apiErr
}

// decodeMessages accepts both ["a", "b"] and "a", which DRF uses
// interchangeably depending on whether the error came from a field validator
// or from a raise inside a view.
func decodeMessages(value json.RawMessage) ([]string, bool) {
	var list []string
	if json.Unmarshal(value, &list) == nil {
		return list, true
	}
	var single string
	if json.Unmarshal(value, &single) == nil {
		return []string{single}, true
	}
	return nil, false
}

func isRetryable(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		// Transport-level failure (connection reset, timeout): worth a retry.
		return true
	}
	switch apiErr.Status {
	case http.StatusTooManyRequests,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func truncate(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	return s[:limit] + "..."
}
