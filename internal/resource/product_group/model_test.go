package product_group

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/schemacommon"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tftest"
)

// The model is assembled from embedded blocks that the framework flattens.
// A missing or misspelled tfsdk tag in a block would otherwise only surface
// during a real apply.
func TestModelMatchesSchema(t *testing.T) {
	before := model{
		Name:        types.StringValue("group"),
		Description: types.StringValue("a group"),
	}
	before.SecurityGateActive = types.BoolValue(true)
	before.SecurityGate.Thresholds = &schemacommon.SecurityGateThresholds{Critical: types.Int64Value(3)}
	before.ObservationNotificationStatusList = types.SetValueMust(
		types.StringType, []attr.Value{types.StringValue("Open")})
	before.AssessmentApprovers = types.SetValueMust(types.Int64Type, []attr.Value{types.Int64Value(7)})
	before.AssessmentApproverAuthorizationGroups = types.SetValueMust(types.Int64Type, nil)
	before.PropagateBranches = []schemacommon.PropagateBranchModel{
		{PropagateTo: types.StringValue("^main$")},
	}

	after := tftest.RoundTrip(t, New(), before)

	if after.Name != before.Name || !after.SecurityGateActive.Equal(before.SecurityGateActive) {
		t.Errorf("round-trip lost values: %+v", after)
	}
}

// A product group created with nothing but a name must still send every
// managed key, otherwise SecObserve's fill-and-clear logic does not fire and
// the result depends on which attributes happened to be set.
func TestToRequestIsComplete(t *testing.T) {
	var diags diag.Diagnostics
	empty := model{Name: types.StringValue("group")}

	request := empty.toRequest(context.Background(), &diags)
	if diags.HasError() {
		t.Fatalf("toRequest: %v", diags.Errors())
	}

	if request.Name != "group" {
		t.Errorf("Name = %q", request.Name)
	}
	// Tri-states stay null so SecObserve keeps inheriting.
	if request.SecurityGateActive != nil {
		t.Errorf("SecurityGateActive = %v, want nil so inheritance is preserved", *request.SecurityGateActive)
	}
	// Collections go out as [] rather than null, which is what the API returns.
	if request.AssessmentApprovers == nil || len(request.AssessmentApprovers) != 0 {
		t.Errorf("AssessmentApprovers = %v, want an empty slice", request.AssessmentApprovers)
	}
	if request.ObservationNotificationStatusList == nil {
		t.Error("ObservationNotificationStatusList = nil, want an empty slice")
	}
}

// Every managed field has to appear in the JSON body. An accidental
// omitempty would silently break the determinism the whole design rests on.
func TestRequestSerializesEveryField(t *testing.T) {
	keys := tftest.JSONKeys(t, client.ProductGroupRequest{})

	for _, required := range []string{
		"name",
		"description",
		"security_gate_active",
		"security_gate_threshold_critical",
		"security_gate_threshold_unknown",
		"repository_branch_housekeeping_active",
		"repository_branch_housekeeping_keep_inactive_days",
		"repository_branch_housekeeping_exempt_branches",
		"notification_ms_teams_webhook",
		"notification_slack_webhook",
		"notification_email_to",
		"observation_notification_min_severity",
		"observation_notification_status_list",
		"observation_notification_min_priority",
		"assessments_need_approval",
		"new_observations_in_review",
		"product_rules_need_approval",
		"assessment_approvers",
		"assessment_approver_authorization_groups",
		"risk_acceptance_expiry_active",
		"risk_acceptance_expiry_days",
		"license_policy",
		"propagate_branches",
		"propagate_branches_new_assessment",
		"propagate_branches_new_observation",
	} {
		if !keys[required] {
			t.Errorf("%q is missing from the serialized request", required)
		}
	}
}
