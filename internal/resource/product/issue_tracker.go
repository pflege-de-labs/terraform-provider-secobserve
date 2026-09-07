package product

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jabbrwcky/terraform-provider-secobserve/internal/client"
	"github.com/jabbrwcky/terraform-provider-secobserve/internal/schemacommon"
	"github.com/jabbrwcky/terraform-provider-secobserve/internal/tfutil"
)

// issueTracker is the issue tracker block. Products only: product groups have
// no issue tracker.
type issueTracker struct {
	IssueTrackerActive          types.Bool   `tfsdk:"issue_tracker_active"`
	IssueTrackerType            types.String `tfsdk:"issue_tracker_type"`
	IssueTrackerBaseURL         types.String `tfsdk:"issue_tracker_base_url"`
	IssueTrackerUsername        types.String `tfsdk:"issue_tracker_username"`
	IssueTrackerAPIKey          types.String `tfsdk:"issue_tracker_api_key"`
	IssueTrackerProjectID       types.String `tfsdk:"issue_tracker_project_id"`
	IssueTrackerLabels          types.String `tfsdk:"issue_tracker_labels"`
	IssueTrackerIssueType       types.String `tfsdk:"issue_tracker_issue_type"`
	IssueTrackerStatusClosed    types.String `tfsdk:"issue_tracker_status_closed"`
	IssueTrackerMinimumSeverity types.String `tfsdk:"issue_tracker_minimum_severity"`
}

func addIssueTracker(attributes map[string]schema.Attribute) {
	attributes["issue_tracker_active"] = schema.BoolAttribute{
		Optional: true,
		Computed: true,
		Default:  booldefault.StaticBool(false),
		MarkdownDescription: "Whether observations are pushed to the issue tracker. Requires " +
			"`issue_tracker_type` to be set.",
	}
	attributes["issue_tracker_type"] = schema.StringAttribute{
		Optional:   true,
		Computed:   true,
		Default:    stringdefault.StaticString(""),
		Validators: []validator.String{stringvalidator.OneOf(schemacommon.WithEmpty(client.IssueTrackerTypes)...)},
		MarkdownDescription: "Which issue tracker to use: " + schemacommon.MarkdownList(client.IssueTrackerTypes) + ".\n\n" +
			"~> SecObserve requires `issue_tracker_type`, `issue_tracker_base_url`, `issue_tracker_api_key` " +
			"and `issue_tracker_project_id` to be **either all set or all empty**.",
	}
	attributes["issue_tracker_base_url"] = schema.StringAttribute{
		Optional:      true,
		Computed:      true,
		Validators:    []validator.String{stringvalidator.LengthAtMost(255)},
		PlanModifiers: []planmodifier.String{schemacommon.StringUnknownWhenStringSiblingChanges("issue_tracker_type")},
		MarkdownDescription: "Base URL of the issue tracker. Left unset with `issue_tracker_type = \"GitHub\"`, " +
			"SecObserve fills in `" + client.GitHubDefaultBaseURL + "`.",
	}
	attributes["issue_tracker_username"] = schema.StringAttribute{
		Optional:   true,
		Computed:   true,
		Default:    stringdefault.StaticString(""),
		Validators: []validator.String{stringvalidator.LengthAtMost(255)},
		MarkdownDescription: "Username for the issue tracker. Required for `Jira` and rejected for every " +
			"other type.",
	}
	attributes["issue_tracker_api_key"] = schema.StringAttribute{
		Optional:   true,
		Computed:   true,
		Sensitive:  true,
		Default:    stringdefault.StaticString(""),
		Validators: []validator.String{stringvalidator.LengthAtMost(255)},
		MarkdownDescription: "API key or token for the issue tracker.\n\n" +
			"~> This value is **stored in Terraform state**. It cannot be a write-only attribute, because " +
			"SecObserve's all-or-none validation over the four core issue tracker fields means every update " +
			"has to resend it. Protect your state accordingly.",
	}
	attributes["issue_tracker_project_id"] = schema.StringAttribute{
		Optional:            true,
		Computed:            true,
		Default:             stringdefault.StaticString(""),
		Validators:          []validator.String{stringvalidator.LengthAtMost(255)},
		MarkdownDescription: "Id or key of the project that issues are created in.",
	}
	attributes["issue_tracker_labels"] = schema.StringAttribute{
		Optional:            true,
		Computed:            true,
		Default:             stringdefault.StaticString(""),
		Validators:          []validator.String{stringvalidator.LengthAtMost(255)},
		MarkdownDescription: "Comma-separated labels to apply to created issues.",
	}
	attributes["issue_tracker_issue_type"] = schema.StringAttribute{
		Optional:   true,
		Computed:   true,
		Default:    stringdefault.StaticString(""),
		Validators: []validator.String{stringvalidator.LengthAtMost(255)},
		MarkdownDescription: "Issue type for created issues. Required for `Jira` and rejected for every other " +
			"type.",
	}
	attributes["issue_tracker_status_closed"] = schema.StringAttribute{
		Optional:   true,
		Computed:   true,
		Default:    stringdefault.StaticString(""),
		Validators: []validator.String{stringvalidator.LengthAtMost(255)},
		MarkdownDescription: "Status that counts as closed. Required for `Jira` and rejected for every other " +
			"type.",
	}
	attributes["issue_tracker_minimum_severity"] = schema.StringAttribute{
		Optional:            true,
		Computed:            true,
		Default:             stringdefault.StaticString(""),
		Validators:          []validator.String{stringvalidator.OneOf(schemacommon.WithEmpty(client.Severities)...)},
		MarkdownDescription: "Only create issues for observations of at least this severity. One of " + schemacommon.MarkdownList(client.Severities) + ".",
	}
}

func (i issueTracker) toAPI() client.IssueTrackerFields {
	return client.IssueTrackerFields{
		IssueTrackerActive:          tfutil.BoolValue(i.IssueTrackerActive),
		IssueTrackerType:            tfutil.StringValue(i.IssueTrackerType),
		IssueTrackerBaseURL:         tfutil.StringValue(i.IssueTrackerBaseURL),
		IssueTrackerUsername:        tfutil.StringValue(i.IssueTrackerUsername),
		IssueTrackerAPIKey:          tfutil.StringValue(i.IssueTrackerAPIKey),
		IssueTrackerProjectID:       tfutil.StringValue(i.IssueTrackerProjectID),
		IssueTrackerLabels:          tfutil.StringValue(i.IssueTrackerLabels),
		IssueTrackerIssueType:       tfutil.StringValue(i.IssueTrackerIssueType),
		IssueTrackerStatusClosed:    tfutil.StringValue(i.IssueTrackerStatusClosed),
		IssueTrackerMinimumSeverity: tfutil.StringValue(i.IssueTrackerMinimumSeverity),
	}
}

func (i *issueTracker) fromAPI(fields client.IssueTrackerFields) {
	i.IssueTrackerActive = types.BoolValue(fields.IssueTrackerActive)
	i.IssueTrackerType = types.StringValue(fields.IssueTrackerType)
	i.IssueTrackerBaseURL = types.StringValue(fields.IssueTrackerBaseURL)
	i.IssueTrackerUsername = types.StringValue(fields.IssueTrackerUsername)
	i.IssueTrackerAPIKey = types.StringValue(fields.IssueTrackerAPIKey)
	i.IssueTrackerProjectID = types.StringValue(fields.IssueTrackerProjectID)
	i.IssueTrackerLabels = types.StringValue(fields.IssueTrackerLabels)
	i.IssueTrackerIssueType = types.StringValue(fields.IssueTrackerIssueType)
	i.IssueTrackerStatusClosed = types.StringValue(fields.IssueTrackerStatusClosed)
	i.IssueTrackerMinimumSeverity = types.StringValue(fields.IssueTrackerMinimumSeverity)
}
