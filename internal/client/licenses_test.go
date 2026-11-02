package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Without an explicit ordering, SecObserve pages a large group's licenses in
// unstable order: pages overlap and licenses go missing.
func TestListLicensesInGroupOrdersStably(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("ordering"); got != "spdx_id" {
			t.Errorf("ordering = %q, want spdx_id", got)
		}
		if got := r.URL.Query().Get("license_groups"); got != "3" {
			t.Errorf("license_groups = %q, want 3", got)
		}
		_ = json.NewEncoder(w).Encode(Page[License]{Results: []License{{ID: 1}}})
	}))
	defer server.Close()

	if _, err := testClient(t, server.URL).ListLicensesInGroup(context.Background(), 3); err != nil {
		t.Fatalf("ListLicensesInGroup: %v", err)
	}
}
