package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr string
	}{
		{"missing base url", Config{APIToken: "t"}, "base_url must not be empty"},
		{"missing token", Config{BaseURL: "https://example.com"}, "api_token must not be empty"},
		{"bad scheme", Config{BaseURL: "ftp://example.com", APIToken: "t"}, "must use http or https"},
		{"ok", Config{BaseURL: "https://example.com/", APIToken: "t"}, ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := New(test.config)
			switch {
			case test.wantErr == "" && err != nil:
				t.Fatalf("unexpected error: %v", err)
			case test.wantErr != "" && err == nil:
				t.Fatalf("expected error containing %q, got nil", test.wantErr)
			case test.wantErr != "" && !strings.Contains(err.Error(), test.wantErr):
				t.Fatalf("expected error containing %q, got %q", test.wantErr, err)
			}
		})
	}
}

// The auth header format is the single most breakage-prone detail of the
// integration: SecObserve uses "APIToken", not "Bearer" or "Token".
func TestAuthorizationHeader(t *testing.T) {
	var gotAuth, gotAccept, gotUserAgent string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotAccept = r.Header.Get("Accept")
		gotUserAgent = r.Header.Get("User-Agent")
		_, _ = w.Write([]byte(`{"version":"1.58.0"}`))
	}))
	defer server.Close()

	c := testClient(t, server.URL)
	if _, err := c.Version(context.Background()); err != nil {
		t.Fatalf("Version: %v", err)
	}

	if want := "APIToken secret-token"; gotAuth != want {
		t.Errorf("Authorization = %q, want %q", gotAuth, want)
	}
	if gotAccept != "application/json" {
		t.Errorf("Accept = %q", gotAccept)
	}
	if !strings.HasPrefix(gotUserAgent, "terraform-provider-secobserve") {
		t.Errorf("User-Agent = %q", gotUserAgent)
	}
}

// The base URL may or may not carry a trailing slash, and callers always pass
// paths without a leading one; neither may produce a doubled or missing slash.
func TestPathJoining(t *testing.T) {
	for _, baseSuffix := range []string{"", "/", "/secobserve", "/secobserve/"} {
		var gotPath, gotQuery string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			gotQuery = r.URL.RawQuery
			_, _ = w.Write([]byte(`{}`))
		}))

		c := testClient(t, server.URL+baseSuffix)
		if err := c.Get(context.Background(), "api/products/1/", map[string][]string{"name": {"a b"}}, &struct{}{}); err != nil {
			t.Fatalf("suffix %q: %v", baseSuffix, err)
		}
		server.Close()

		wantPath := strings.TrimSuffix(baseSuffix, "/") + "/api/products/1/"
		if gotPath != wantPath {
			t.Errorf("suffix %q: path = %q, want %q", baseSuffix, gotPath, wantPath)
		}
		if gotQuery != "name=a+b" {
			t.Errorf("suffix %q: query = %q", baseSuffix, gotQuery)
		}
	}
}

// 204 is the success response for approvals and several bulk actions; decoding
// an empty body as JSON would fail.
func TestNoContentIsNotDecoded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	out := struct{ Name string }{Name: "untouched"}
	if err := testClient(t, server.URL).Get(context.Background(), "api/x/", nil, &out); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if out.Name != "untouched" {
		t.Errorf("output was modified: %+v", out)
	}
}

func TestRetryOnThrottling(t *testing.T) {
	var calls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{"version":"1.58.0"}`))
	}))
	defer server.Close()

	version, err := testClient(t, server.URL).Version(context.Background())
	if err != nil {
		t.Fatalf("Version: %v", err)
	}
	if version.Version != "1.58.0" {
		t.Errorf("version = %q", version.Version)
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("calls = %d, want 2", got)
	}
}

// A 400 is a configuration error: retrying it would only slow down the failure.
func TestNoRetryOnClientError(t *testing.T) {
	var calls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"name":["This field is required."]}`))
	}))
	defer server.Close()

	if _, err := testClient(t, server.URL).Version(context.Background()); err == nil {
		t.Fatal("expected an error")
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("calls = %d, want 1", got)
	}
}

func TestContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	if _, err := testClient(t, server.URL).Version(ctx); err == nil {
		t.Fatal("expected an error")
	}
}

func testClient(t *testing.T, baseURL string) *Client {
	t.Helper()
	c, err := New(Config{BaseURL: baseURL, APIToken: "secret-token"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}
