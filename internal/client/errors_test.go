package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Django REST Framework does not have one error shape. These are the four the
// SecObserve backend actually produces.
func TestAPIErrorParsing(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		wantDetail string
		wantFields map[string][]string
		wantInMsg  string
	}{
		{
			name:       "field validation errors",
			status:     http.StatusBadRequest,
			body:       `{"name":["This field is required."],"role":["Ensure this value is less than or equal to 5."]}`,
			wantFields: map[string][]string{"name": {"This field is required."}, "role": {"Ensure this value is less than or equal to 5."}},
			wantInMsg:  "name: This field is required., role: Ensure this value",
		},
		{
			name:       "permission denied detail",
			status:     http.StatusForbidden,
			body:       `{"detail":"You do not have permission to perform this action."}`,
			wantDetail: "You do not have permission to perform this action.",
		},
		{
			// The SecObserve exception handler turns a protected foreign key
			// into a 409 with "message" rather than DRF's "detail".
			name:       "protected relation conflict",
			status:     http.StatusConflict,
			body:       `{"message":"Cannot delete product because it is referenced by a license policy"}`,
			wantDetail: "Cannot delete product because it is referenced by a license policy",
		},
		{
			// Errors raised inside a view arrive as a bare string, not a list.
			name:       "single string field error",
			status:     http.StatusBadRequest,
			body:       `{"name":"Confirmation name must match the product name."}`,
			wantFields: map[string][]string{"name": {"Confirmation name must match the product name."}},
		},
		{
			name:      "unparseable body is preserved",
			status:    http.StatusBadGateway,
			body:      `<html>502 Bad Gateway</html>`,
			wantInMsg: "502 Bad Gateway",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.body))
			}))
			defer server.Close()

			// Disable retries for the retryable statuses by using a cancelled
			// budget: we only care about the final error's contents.
			err := testClient(t, server.URL).Do(context.Background(), Request{Method: http.MethodGet, Path: "api/x/"})
			if err == nil {
				t.Fatal("expected an error")
			}

			apiErr := asAPIError(t, err)
			if apiErr.Status != test.status {
				t.Errorf("Status = %d, want %d", apiErr.Status, test.status)
			}
			if apiErr.Detail != test.wantDetail {
				t.Errorf("Detail = %q, want %q", apiErr.Detail, test.wantDetail)
			}
			for field, want := range test.wantFields {
				got := apiErr.FieldErrors[field]
				if len(got) != len(want) || (len(want) > 0 && got[0] != want[0]) {
					t.Errorf("FieldErrors[%q] = %v, want %v", field, got, want)
				}
			}
			if test.wantInMsg != "" && !strings.Contains(apiErr.Error(), test.wantInMsg) {
				t.Errorf("message %q does not contain %q", apiErr.Error(), test.wantInMsg)
			}
		})
	}
}

func TestErrorClassifiers(t *testing.T) {
	tests := []struct {
		status                               int
		notFound, conflict, forbidden, retry bool
	}{
		{http.StatusNotFound, true, false, false, false},
		{http.StatusConflict, false, true, false, false},
		{http.StatusForbidden, false, false, true, false},
		{http.StatusBadRequest, false, false, false, false},
		{http.StatusTooManyRequests, false, false, false, true},
		{http.StatusServiceUnavailable, false, false, false, true},
		{http.StatusInternalServerError, false, false, false, false},
	}

	for _, test := range tests {
		err := &APIError{Status: test.status}
		if got := NotFound(err); got != test.notFound {
			t.Errorf("status %d: NotFound = %v", test.status, got)
		}
		if got := Conflict(err); got != test.conflict {
			t.Errorf("status %d: Conflict = %v", test.status, got)
		}
		if got := Forbidden(err); got != test.forbidden {
			t.Errorf("status %d: Forbidden = %v", test.status, got)
		}
		if got := isRetryable(err); got != test.retry {
			t.Errorf("status %d: isRetryable = %v", test.status, got)
		}
	}
}

// A transport failure has no status and must stay retryable.
func TestTransportErrorIsRetryable(t *testing.T) {
	if !isRetryable(context.DeadlineExceeded) {
		t.Error("transport errors should be retryable")
	}
}

func asAPIError(t *testing.T, err error) *APIError {
	t.Helper()
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	return apiErr
}
