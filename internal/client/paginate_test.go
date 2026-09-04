package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestListFollowsAllPages(t *testing.T) {
	var seenPages []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		seenPages = append(seenPages, page)

		if got := r.URL.Query().Get("page_size"); got != "100" {
			t.Errorf("page_size = %q, want 100", got)
		}
		if got := r.URL.Query().Get("product"); got != "7" {
			t.Errorf("caller filter was dropped: product = %q", got)
		}

		body := Page[Parser]{}
		switch page {
		case "1":
			next := server(r) + "?page=2"
			body = Page[Parser]{Count: 3, Next: &next, Results: []Parser{{ID: 1, Name: "a"}, {ID: 2, Name: "b"}}}
		case "2":
			body = Page[Parser]{Count: 3, Results: []Parser{{ID: 3, Name: "c"}}}
		default:
			t.Errorf("unexpected page %q", page)
		}
		_ = json.NewEncoder(w).Encode(body)
	}))
	defer server.Close()

	got, err := List[Parser](context.Background(), testClient(t, server.URL), "api/parsers/", url.Values{"product": {"7"}})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d items, want 3: %+v", len(got), got)
	}
	if len(seenPages) != 2 {
		t.Errorf("requested pages %v, want two", seenPages)
	}
}

// A truthful "count" with a null "next" ends the walk: SecObserve's count can
// change mid-walk on a live instance, so it must not drive termination.
func TestListStopsOnNilNextDespiteCount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(Page[Parser]{Count: 9999, Next: nil, Results: []Parser{{ID: 1}}})
	}))
	defer server.Close()

	got, err := List[Parser](context.Background(), testClient(t, server.URL), "api/parsers/", nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("got %d items, want 1", len(got))
	}
}

// An empty page also terminates, guarding against a backend that keeps
// reporting a "next" link.
func TestListStopsOnEmptyPage(t *testing.T) {
	next := "http://example.invalid/?page=2"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(Page[Parser]{Count: 1, Next: &next, Results: nil})
	}))
	defer server.Close()

	got, err := List[Parser](context.Background(), testClient(t, server.URL), "api/parsers/", nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d items, want 0", len(got))
	}
}

// /api/product_api_tokens/ is a plain ViewSet and answers with a bare results
// envelope, without count or next.
func TestListUnpaginated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `{"results":[{"id":4,"name":"ci"}]}`)
	}))
	defer server.Close()

	got, err := ListUnpaginated[Parser](context.Background(), testClient(t, server.URL), "api/product_api_tokens/", nil)
	if err != nil {
		t.Fatalf("ListUnpaginated: %v", err)
	}
	if len(got) != 1 || got[0].ID != 4 {
		t.Errorf("got %+v", got)
	}
}

// server reconstructs the request's own base URL so a handler can build a
// plausible "next" link.
func server(r *http.Request) string {
	return "http://" + r.Host + r.URL.Path
}
