package product

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jabbrwcky/terraform-provider-secobserve/internal/client"
	"github.com/jabbrwcky/terraform-provider-secobserve/internal/schemacommon"
	"github.com/jabbrwcky/terraform-provider-secobserve/internal/tftest"
)

// The product model is assembled from eight shared blocks plus two
// product-only ones. A missing or misspelled tfsdk tag in any of them would
// otherwise only surface during a real apply.
func TestModelMatchesSchema(t *testing.T) {
	before := model{
		Name:        types.StringValue("product"),
		Description: types.StringValue("a product"),
	}
	before.SecurityGateActive = types.BoolValue(true)
	before.SecurityGateThresholdCritical = types.Int64Value(0)
	before.IssueTrackerType = types.StringValue("Jira")
	before.OSVLinuxDistribution = types.StringValue("Debian")
	before.ObservationNotificationStatusList = types.SetValueMust(
		types.StringType, []attr.Value{types.StringValue("Open")})
	before.AssessmentApprovers = types.SetValueMust(types.Int64Type, []attr.Value{types.Int64Value(7)})
	before.AssessmentApproverAuthorizationGroups = types.SetValueMust(types.Int64Type, nil)
	before.PropagateBranches = []schemacommon.PropagateBranchModel{
		{PropagateTo: types.StringValue("^main$")},
	}

	after := tftest.RoundTrip(t, New(), before)

	if after.Name != before.Name ||
		!after.IssueTrackerType.Equal(before.IssueTrackerType) ||
		!after.OSVLinuxDistribution.Equal(before.OSVLinuxDistribution) {
		t.Errorf("round-trip lost values: %+v", after)
	}
}

// Every managed field has to appear in the JSON body: SecObserve reads an
// absent key as "leave unchanged", and its fill-and-clear logic for the
// security gate and branch housekeeping only fires when the matching *_active
// flag is present. An accidental omitempty would silently break that.
func TestRequestSerializesEveryField(t *testing.T) {
	keys := tftest.JSONKeys(t, client.ProductRequest{})

	for _, required := range []string{
		"name", "description", "product_group", "purl", "cpe23", "repository_prefix",
		"apply_general_rules",
		"security_gate_active",
		"security_gate_threshold_critical", "security_gate_threshold_high",
		"security_gate_threshold_medium", "security_gate_threshold_low",
		"security_gate_threshold_none", "security_gate_threshold_unknown",
		"repository_branch_housekeeping_active",
		"repository_branch_housekeeping_keep_inactive_days",
		"repository_branch_housekeeping_exempt_branches",
		"notification_ms_teams_webhook", "notification_slack_webhook", "notification_email_to",
		"observation_notification_min_severity", "observation_notification_status_list",
		"observation_notification_min_priority",
		"assessments_need_approval", "new_observations_in_review", "product_rules_need_approval",
		"assessment_approvers", "assessment_approver_authorization_groups",
		"risk_acceptance_expiry_active", "risk_acceptance_expiry_days",
		"license_policy",
		"propagate_branches", "propagate_branches_new_assessment", "propagate_branches_new_observation",
		"issue_tracker_active", "issue_tracker_type", "issue_tracker_base_url",
		"issue_tracker_username", "issue_tracker_api_key", "issue_tracker_project_id",
		"issue_tracker_labels", "issue_tracker_issue_type", "issue_tracker_status_closed",
		"issue_tracker_minimum_severity",
		"osv_enabled", "osv_linux_distribution", "osv_linux_release", "automatic_osv_scanning_enabled",
		"vulnerablecode_enabled", "automatic_vulnerablecode_scanning_enabled",
	} {
		if !keys[required] {
			t.Errorf("%q is missing from the serialized request", required)
		}
	}

	// Read-only fields must never be written back.
	for _, forbidden := range []string{
		"id", "is_product_group", "security_gate_passed", "repository_default_branch",
		"product_group_name", "repository_default_branch_name",
	} {
		if keys[forbidden] {
			t.Errorf("%q must not be part of a write request", forbidden)
		}
	}
}

// The tri-states are the reason this provider uses the plugin framework: null
// has to stay distinguishable from false so SecObserve keeps inheriting.
func TestTriStatesStayNull(t *testing.T) {
	var diags diag.Diagnostics
	request := model{Name: types.StringValue("product")}.toRequest(context.Background(), &diags)
	if diags.HasError() {
		t.Fatalf("toRequest: %v", diags.Errors())
	}

	if request.SecurityGateActive != nil {
		t.Errorf("SecurityGateActive = %v, want nil", *request.SecurityGateActive)
	}
	if request.RepositoryBranchHousekeepingActive != nil {
		t.Errorf("RepositoryBranchHousekeepingActive = %v, want nil", *request.RepositoryBranchHousekeepingActive)
	}
	if request.RiskAcceptanceExpiryActive != nil {
		t.Errorf("RiskAcceptanceExpiryActive = %v, want nil", *request.RiskAcceptanceExpiryActive)
	}
}
