package product

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tfutil"
)

// ValidateConfig mirrors SecObserve's cross-field validation so the errors
// appear at plan time rather than as a 400 halfway through an apply.
func (r *productResource) ValidateConfig(
	ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse,
) {
	var config model
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	config.SecurityGate.ValidateSecurityGate(&resp.Diagnostics)
	config.BranchPropagation.ValidateBranchPropagation(&resp.Diagnostics)
	config.validateIssueTracker(&resp.Diagnostics)
	config.validateScanners(&resp.Diagnostics)
}

// validateIssueTracker covers the all-or-none rule, the requirement that an
// active tracker has a type, and the Jira-only fields.
func (m model) validateIssueTracker(diags *diag.Diagnostics) {
	if m.IssueTrackerType.IsUnknown() || m.IssueTrackerBaseURL.IsUnknown() ||
		m.IssueTrackerAPIKey.IsUnknown() || m.IssueTrackerProjectID.IsUnknown() {
		// A core field resolves to unknown when it comes from a variable or
		// resource attribute not yet known (e.g. every root variable is
		// unknown during `terraform validate`, regardless of whether it has
		// a value). Treating unknown the same as absent would misreport a
		// configured value as missing; defer the all-or-none check to
		// SecObserve's own 400 as the backstop once the value is known.
		return
	}

	trackerType := tfutil.StringValue(m.IssueTrackerType)
	baseURL := tfutil.StringValue(m.IssueTrackerBaseURL)
	apiKey := tfutil.StringValue(m.IssueTrackerAPIKey)
	projectID := tfutil.StringValue(m.IssueTrackerProjectID)

	// GitHub is the one type whose base URL SecObserve fills in, so an unset
	// base URL is not a gap there.
	baseURLSatisfied := baseURL != "" || trackerType == client.IssueTrackerGitHub

	core := []struct {
		attribute string
		set       bool
	}{
		{"issue_tracker_type", trackerType != ""},
		{"issue_tracker_base_url", baseURLSatisfied},
		{"issue_tracker_api_key", apiKey != ""},
		{"issue_tracker_project_id", projectID != ""},
	}

	setCount := 0
	for _, field := range core {
		if field.set {
			setCount++
		}
	}
	if setCount != 0 && setCount != len(core) {
		missing := make([]string, 0, len(core))
		for _, field := range core {
			if !field.set {
				missing = append(missing, field.attribute)
			}
		}
		diags.AddError(
			"Incomplete issue tracker configuration",
			"SecObserve requires issue_tracker_type, issue_tracker_base_url, issue_tracker_api_key and "+
				"issue_tracker_project_id to be either all set or all empty. Missing: "+
				strings.Join(missing, ", ")+".\n\n"+
				"With issue_tracker_type = \"GitHub\" the base URL may be omitted; SecObserve fills in "+
				client.GitHubDefaultBaseURL+".",
		)
	}

	if tfutil.BoolValue(m.IssueTrackerActive) && trackerType == "" {
		diags.AddAttributeError(
			path.Root("issue_tracker_active"),
			"Issue tracker is active but not configured",
			"Set issue_tracker_type and the rest of the issue tracker attributes, or set "+
				"issue_tracker_active to false.",
		)
	}

	jiraOnly := []struct {
		attribute string
		value     string
	}{
		{"issue_tracker_username", tfutil.StringValue(m.IssueTrackerUsername)},
		{"issue_tracker_issue_type", tfutil.StringValue(m.IssueTrackerIssueType)},
		{"issue_tracker_status_closed", tfutil.StringValue(m.IssueTrackerStatusClosed)},
	}

	switch {
	case trackerType == client.IssueTrackerJira:
		for _, field := range jiraOnly {
			if field.value == "" {
				diags.AddAttributeError(
					path.Root(field.attribute),
					"Attribute is required for Jira",
					"SecObserve requires "+field.attribute+" when issue_tracker_type is \"Jira\".",
				)
			}
		}
	case trackerType != "":
		for _, field := range jiraOnly {
			if field.value != "" {
				diags.AddAttributeError(
					path.Root(field.attribute),
					"Attribute is only valid for Jira",
					"SecObserve rejects "+field.attribute+" unless issue_tracker_type is \"Jira\", but "+
						"the type is \""+trackerType+"\".",
				)
			}
		}
	}
}

func (m model) validateScanners(diags *diag.Diagnostics) {
	if m.OSVLinuxRelease.IsUnknown() || m.OSVLinuxDistribution.IsUnknown() {
		// See the matching comment in validateIssueTracker: unknown must not
		// be treated as absent.
		return
	}

	if tfutil.StringValue(m.OSVLinuxRelease) != "" && tfutil.StringValue(m.OSVLinuxDistribution) == "" {
		diags.AddAttributeError(
			path.Root("osv_linux_release"),
			"osv_linux_release needs a distribution",
			"SecObserve rejects osv_linux_release unless osv_linux_distribution is also set.",
		)
	}
}
