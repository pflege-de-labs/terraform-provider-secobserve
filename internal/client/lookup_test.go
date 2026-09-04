package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// SecObserve filters names with icontains, so the server happily returns
// near-matches. Resolving "Trivy" must not silently pick "Trivy Operator".
func TestFindByExactNameRejectsSubstringMatches(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("name"); got != "Trivy" {
			t.Errorf("name filter = %q", got)
		}
		_ = json.NewEncoder(w).Encode(Page[Parser]{
			Count:   2,
			Results: []Parser{{ID: 1, Name: "Trivy Operator Prometheus"}, {ID: 2, Name: "Trivy Filesystem"}},
		})
	}))
	defer server.Close()

	_, err := FindByExactName[Parser](context.Background(), testClient(t, server.URL), "parser", "api/parsers/", "Trivy", nil)

	var notFound *ErrNotFound
	if !errors.As(err, &notFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if !strings.Contains(notFound.Error(), `no parser found with name "Trivy"`) {
		t.Errorf("message = %q", notFound.Error())
	}
}

func TestFindByExactNameReturnsSingleMatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(Page[Parser]{
			Count:   2,
			Results: []Parser{{ID: 1, Name: "Trivy Operator Prometheus"}, {ID: 2, Name: "Trivy"}},
		})
	}))
	defer server.Close()

	got, err := FindByExactName[Parser](context.Background(), testClient(t, server.URL), "parser", "api/parsers/", "Trivy", nil)
	if err != nil {
		t.Fatalf("FindByExactName: %v", err)
	}
	if got.ID != 2 {
		t.Errorf("id = %d, want 2", got.ID)
	}
}

// General rule names are not uniquely constrained in the database, so an
// ambiguous result has to be reported rather than resolved arbitrarily:
// picking one would silently manage the wrong object.
func TestFindByExactNameReportsAmbiguity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(Page[Parser]{
			Count:   2,
			Results: []Parser{{ID: 7, Name: "duplicate"}, {ID: 9, Name: "duplicate"}},
		})
	}))
	defer server.Close()

	_, err := FindByExactName[Parser](context.Background(), testClient(t, server.URL), "general rule", "api/general_rules/", "duplicate", nil)

	var ambiguous *ErrAmbiguous
	if !errors.As(err, &ambiguous) {
		t.Fatalf("expected ErrAmbiguous, got %v", err)
	}
	if len(ambiguous.IDs) != 2 {
		t.Errorf("IDs = %v", ambiguous.IDs)
	}
	if !strings.Contains(ambiguous.Error(), "reference the object by id") {
		t.Errorf("message = %q", ambiguous.Error())
	}
}

// Name matching is case-sensitive, matching the backend's own delete
// confirmation semantics.
func TestFindByExactNameIsCaseSensitive(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(Page[Parser]{Count: 1, Results: []Parser{{ID: 1, Name: "trivy"}}})
	}))
	defer server.Close()

	_, err := FindByExactName[Parser](context.Background(), testClient(t, server.URL), "parser", "api/parsers/", "Trivy", nil)
	var notFound *ErrNotFound
	if !errors.As(err, &notFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestMajorMinor(t *testing.T) {
	tests := map[string]string{
		"1.58.0":       "1.58",
		"v1.58.3":      "1.58",
		"1.58":         "1.58",
		"2.0.0":        "2.0",
		UnknownVersion: UnknownVersion,
	}
	for input, want := range tests {
		if got := MajorMinor(input); got != want {
			t.Errorf("MajorMinor(%q) = %q, want %q", input, got, want)
		}
	}
}
