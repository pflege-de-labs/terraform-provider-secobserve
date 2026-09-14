package schemacommon

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jabbrwcky/terraform-provider-secobserve/internal/client"
	"github.com/jabbrwcky/terraform-provider-secobserve/internal/tfutil"
)

// Approvals is the review and approval block of a product or product group.
type Approvals struct {
	AssessmentsNeedApproval  types.Bool `tfsdk:"assessments_need_approval"`
	NewObservationsInReview  types.Bool `tfsdk:"new_observations_in_review"`
	ProductRulesNeedApproval types.Bool `tfsdk:"product_rules_need_approval"`
}

// Approvers designates who may approve assessments.
type Approvers struct {
	AssessmentApprovers                   types.Set `tfsdk:"assessment_approvers"`
	AssessmentApproverAuthorizationGroups types.Set `tfsdk:"assessment_approver_authorization_groups"`
}

// AddApprovals contributes the approval workflow attributes.
func AddApprovals(attributes map[string]schema.Attribute) {
	attributes["assessments_need_approval"] = schema.BoolAttribute{
		Optional: true,
		Computed: true,
		Default:  booldefault.StaticBool(false),
		MarkdownDescription: "Require a second person to approve assessments of observations.\n\n" +
			"~> SecObserve refuses to switch this off again while assessments are still pending approval. " +
			"Approve, reject or delete them first.",
	}
	attributes["new_observations_in_review"] = schema.BoolAttribute{
		Optional:            true,
		Computed:            true,
		Default:             booldefault.StaticBool(false),
		MarkdownDescription: "Put newly imported observations into the `In review` status.",
	}
	attributes["product_rules_need_approval"] = schema.BoolAttribute{
		Optional: true,
		Computed: true,
		Default:  booldefault.StaticBool(false),
		MarkdownDescription: "Require approval for product rules.\n\n" +
			"~> A rule created while this is enabled stays inactive until somebody **other than its author** " +
			"approves it, and SecObserve rejects self-approval. Since Terraform creates rules as the provider's " +
			"own identity, approval always has to happen outside Terraform.",
	}
}

// AddApprovers contributes the assessment approver attributes.
func AddApprovers(attributes map[string]schema.Attribute, scope string) {
	attributes["assessment_approvers"] = schema.SetAttribute{
		Optional:    true,
		Computed:    true,
		ElementType: types.Int64Type,
		Default:     setdefault.StaticValue(EmptyInt64Set()),
		Validators:  []validator.Set{setvalidator.ValueInt64sAre(int64validator.AtLeast(1))},
		MarkdownDescription: "Ids of the users designated to approve assessments. Use the `secobserve_user` " +
			"data source to resolve usernames to ids.",
	}
	attributes["assessment_approver_authorization_groups"] = schema.SetAttribute{
		Optional:    true,
		Computed:    true,
		ElementType: types.Int64Type,
		Default:     setdefault.StaticValue(EmptyInt64Set()),
		Validators:  []validator.Set{setvalidator.ValueInt64sAre(int64validator.AtLeast(1))},
		MarkdownDescription: "Ids of the authorization groups designated to approve assessments.\n\n" +
			"~> Each group must **already** hold at least the `Writer` role on this " + scope + ", otherwise " +
			"SecObserve rejects the request. Declare the corresponding " +
			"`secobserve_product_authorization_group_member` first and reference its " +
			"`authorization_group` so Terraform orders them correctly.",
	}
}

// ToAPI converts the block into its request representation.
func (a Approvals) ToAPI() client.ApprovalFields {
	return client.ApprovalFields{
		AssessmentsNeedApproval:  tfutil.BoolValue(a.AssessmentsNeedApproval),
		NewObservationsInReview:  tfutil.BoolValue(a.NewObservationsInReview),
		ProductRulesNeedApproval: tfutil.BoolValue(a.ProductRulesNeedApproval),
	}
}

// FromAPI fills the block from an API response.
func (a *Approvals) FromAPI(fields client.ApprovalFields) {
	a.AssessmentsNeedApproval = types.BoolValue(fields.AssessmentsNeedApproval)
	a.NewObservationsInReview = types.BoolValue(fields.NewObservationsInReview)
	a.ProductRulesNeedApproval = types.BoolValue(fields.ProductRulesNeedApproval)
}

// ToAPI converts the block into its request representation.
func (a Approvers) ToAPI(ctx context.Context, diags *diag.Diagnostics) client.ApproverFields {
	return client.ApproverFields{
		AssessmentApprovers: emptyInt64IfNil(
			tfutil.Int64Set(ctx, a.AssessmentApprovers, diags)),
		AssessmentApproverAuthorizationGroups: emptyInt64IfNil(
			tfutil.Int64Set(ctx, a.AssessmentApproverAuthorizationGroups, diags)),
	}
}

// FromAPI fills the block from an API response.
func (a *Approvers) FromAPI(ctx context.Context, fields client.ApproverFields, diags *diag.Diagnostics) {
	a.AssessmentApprovers = tfutil.Int64SetValue(ctx, fields.AssessmentApprovers, diags)
	a.AssessmentApproverAuthorizationGroups = tfutil.Int64SetValue(ctx, fields.AssessmentApproverAuthorizationGroups, diags)
}

func emptyInt64IfNil(values []int64) []int64 {
	if values == nil {
		return []int64{}
	}
	return values
}
