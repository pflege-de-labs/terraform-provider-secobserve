package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Generated config for a group listed as a product would fight
// secobserve_product_group, so the filter must hold whatever the endpoint returns.
func TestProductsExcludesGroups(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/products/" {
			t.Errorf("path = %q, want /api/products/", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(Page[Product]{Results: []Product{
			{ID: 1, Name: "p", IsProductGroup: false},
			{ID: 2, Name: "g", IsProductGroup: true},
		}})
	}))
	defer server.Close()

	got, err := testClient(t, server.URL).Products(context.Background())
	if err != nil {
		t.Fatalf("Products: %v", err)
	}
	if len(got) != 1 || got[0].ID != 1 {
		t.Fatalf("got %+v, want only product 1", got)
	}
}
