package product

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Regression test: an unknown config value (e.g. issue_tracker_api_key set
// from a Terraform variable, which `terraform validate` always resolves to
// unknown regardless of whether it has a value) must not be treated the same
// as an absent one. Before the fix, tfutil.StringValue mapped unknown to ""
// and the all-or-none check reported issue_tracker_api_key as missing even
// though it was actually configured.
func TestValidateIssueTrackerDoesNotFlagUnknownAPIKeyAsMissing(t *testing.T) {
	var m model
	m.IssueTrackerType = types.StringValue("GitHub")
	m.IssueTrackerAPIKey = types.StringUnknown()
	m.IssueTrackerProjectID = types.StringValue("pflege-de-labs/teamster")

	var diags diag.Diagnostics
	m.validateIssueTracker(&diags)

	if diags.HasError() {
		t.Fatalf("expected no error for an unknown issue_tracker_api_key, got: %v", diags)
	}
}

func TestValidateIssueTrackerStillRejectsGenuinelyMissingAPIKey(t *testing.T) {
	var m model
	m.IssueTrackerType = types.StringValue("GitHub")
	m.IssueTrackerAPIKey = types.StringValue("")
	m.IssueTrackerProjectID = types.StringValue("pflege-de-labs/teamster")

	var diags diag.Diagnostics
	m.validateIssueTracker(&diags)

	if !diags.HasError() {
		t.Fatal("expected an error for a genuinely empty issue_tracker_api_key")
	}
}

func TestValidateScannersDoesNotFlagUnknownOSVDistributionAsMissing(t *testing.T) {
	var m model
	m.OSVLinuxRelease = types.StringValue("12")
	m.OSVLinuxDistribution = types.StringUnknown()

	var diags diag.Diagnostics
	m.validateScanners(&diags)

	if diags.HasError() {
		t.Fatalf("expected no error for an unknown osv_linux_distribution, got: %v", diags)
	}
}
