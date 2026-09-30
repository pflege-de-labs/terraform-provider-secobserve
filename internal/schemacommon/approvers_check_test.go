package schemacommon

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
)

// Memberships of authorization group 7: Writer on product group 10, Reader on product 20.
const approverMembersBody = `{"count":2,"next":null,"previous":null,"results":[
	{"id":1,"product":10,"authorization_group":7,"role":3},
	{"id":2,"product":20,"authorization_group":7,"role":1}]}`

func approverTestClient(t *testing.T) *client.Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("authorization_group") == "7" {
			_, _ = w.Write([]byte(approverMembersBody))
			return
		}
		_, _ = w.Write([]byte(`{"count":0,"next":null,"previous":null,"results":[]}`))
	}))
	t.Cleanup(server.Close)

	c, err := client.New(client.Config{BaseURL: server.URL, APIToken: "token"})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func approverGroups(values ...attr.Value) Approvers {
	return Approvers{AssessmentApproverAuthorizationGroups: types.SetValueMust(types.Int64Type, values)}
}

func TestCheckApproverGroups(t *testing.T) {
	c := approverTestClient(t)

	for _, tc := range []struct {
		name       string
		approvers  Approvers
		productIDs []int64
		client     *client.Client
		wantError  string
		wantWarn   string
	}{
		{name: "no approvers", approvers: approverGroups(), client: c},
		{
			name:      "create without anything to hold a membership",
			approvers: approverGroups(types.Int64Value(7)),
			client:    c,
			wantError: "Approver groups cannot be set on create",
		},
		{
			name:       "Writer on the product group",
			approvers:  approverGroups(types.Int64Value(7)),
			productIDs: []int64{10},
			client:     c,
		},
		{
			name:       "only Reader on the product",
			approvers:  approverGroups(types.Int64Value(7)),
			productIDs: []int64{20},
			client:     c,
			wantWarn:   "Approver group lacks the Writer role",
		},
		{
			name:       "no membership at all",
			approvers:  approverGroups(types.Int64Value(8)),
			productIDs: []int64{10},
			client:     c,
			wantWarn:   "Approver group lacks the Writer role",
		},
		{
			name:       "unknown group id defers to apply",
			approvers:  approverGroups(types.Int64Unknown()),
			productIDs: nil,
			client:     c,
		},
		{
			name:       "unconfigured client skips the lookup",
			approvers:  approverGroups(types.Int64Value(8)),
			productIDs: []int64{10},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var diags diag.Diagnostics
			tc.approvers.CheckApproverGroups(context.Background(), tc.client, tc.productIDs, &diags)

			assertDiag(t, diags.Errors(), tc.wantError)
			assertDiag(t, diags.Warnings(), tc.wantWarn)
		})
	}
}

func assertDiag(t *testing.T, diags diag.Diagnostics, want string) {
	t.Helper()
	if want == "" {
		if len(diags) != 0 {
			t.Errorf("unexpected diagnostics: %v", diags)
		}
		return
	}
	if len(diags) != 1 || !strings.Contains(diags[0].Summary(), want) {
		t.Errorf("want one diagnostic %q, got %v", want, diags)
	}
}
