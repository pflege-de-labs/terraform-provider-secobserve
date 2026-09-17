package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
)

func TestVerifyInstance(t *testing.T) {
	tests := []struct {
		name        string
		me          string
		meStatus    int
		version     string
		wantErrors  []string
		wantWarns   []string
		wantNoWarns bool
	}{
		{
			name:        "superuser on the expected version",
			me:          `{"id":1,"username":"admin","is_superuser":true}`,
			version:     `{"version":"` + client.SchemaVersion + `"}`,
			wantNoWarns: true,
		},
		{
			name:        "patch releases are compatible",
			me:          `{"id":1,"username":"admin","is_superuser":true}`,
			version:     `{"version":"1.59.99"}`,
			wantNoWarns: true,
		},
		{
			// An instance running from source never gets its version
			// substituted into views.py, so this is not a mismatch.
			name:        "unreported version is not a mismatch",
			me:          `{"id":1,"username":"admin","is_superuser":true}`,
			version:     `{"version":"` + client.UnknownVersion + `"}`,
			wantNoWarns: true,
		},
		{
			name:      "non-superuser is warned about",
			me:        `{"id":2,"username":"ci-user","is_superuser":false}`,
			version:   `{"version":"` + client.SchemaVersion + `"}`,
			wantWarns: []string{"does not belong to a superuser", "ci-user"},
		},
		{
			name:      "minor version drift is warned about",
			me:        `{"id":1,"username":"admin","is_superuser":true}`,
			version:   `{"version":"1.42.0"}`,
			wantWarns: []string{"differs from the version this provider was built against", "1.42.0"},
		},
		{
			name:       "rejected token is an error",
			meStatus:   http.StatusForbidden,
			me:         `{"detail":"Invalid token."}`,
			wantErrors: []string{"Could not authenticate against SecObserve"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/users/me/":
					if test.meStatus != 0 {
						w.WriteHeader(test.meStatus)
					}
					_, _ = w.Write([]byte(test.me))
				case "/api/status/version/":
					_, _ = w.Write([]byte(test.version))
				default:
					t.Errorf("unexpected request to %s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()

			apiClient, err := client.New(client.Config{BaseURL: server.URL, APIToken: "t"})
			if err != nil {
				t.Fatalf("client.New: %v", err)
			}

			var diags diag.Diagnostics
			verifyInstance(context.Background(), apiClient, &diags)

			errorText := joinSummaries(diags.Errors())
			for _, want := range test.wantErrors {
				if !strings.Contains(errorText, want) {
					t.Errorf("errors %q do not contain %q", errorText, want)
				}
			}
			if len(test.wantErrors) == 0 && diags.HasError() {
				t.Errorf("unexpected errors: %q", errorText)
			}

			warningText := joinSummaries(diags.Warnings())
			for _, want := range test.wantWarns {
				if !strings.Contains(warningText, want) {
					t.Errorf("warnings %q do not contain %q", warningText, want)
				}
			}
			if test.wantNoWarns && len(diags.Warnings()) > 0 {
				t.Errorf("unexpected warnings: %q", warningText)
			}
		})
	}
}

func joinSummaries(diags diag.Diagnostics) string {
	var parts []string
	for _, d := range diags {
		parts = append(parts, d.Summary(), d.Detail())
	}
	return strings.Join(parts, " | ")
}
