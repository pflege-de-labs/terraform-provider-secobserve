package schemacommon

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tfutil"
	sovalidators "github.com/pflege-de-labs/terraform-provider-secobserve/internal/validators"
)

// BranchPropagation is the branch propagation block of a product or product
// group.
//
// PropagateBranches is types.List, not a plain Go slice: a config expression
// whose result depends on a not-yet-known upstream value (e.g. a for_each
// over a map derived from another resource) plans this attribute as unknown,
// and the framework's reflection-based Get/Set cannot populate a plain slice
// field from an unknown list -- it panics with "Received unknown value,
// however the target type cannot handle unknown values", naming
// basetypes.ListValue as the fix. ElementsAs/ListValueFrom convert to/from
// []PropagateBranchModel explicitly in ToAPI/FromAPI instead.
type BranchPropagation struct {
	PropagateBranches               types.List `tfsdk:"propagate_branches"`
	PropagateBranchesNewAssessment  types.Bool `tfsdk:"propagate_branches_new_assessment"`
	PropagateBranchesNewObservation types.Bool `tfsdk:"propagate_branches_new_observation"`
}

// PropagateBranchModel is one branch propagation rule.
type PropagateBranchModel struct {
	PropagateTo types.String `tfsdk:"propagate_to"`
}

// PropagateBranchObjectType is PropagateBranchModel's object type, needed to
// build/inspect PropagateBranches without a known Go slice on hand.
var PropagateBranchObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{"propagate_to": types.StringType},
}

// AddBranchPropagation contributes the branch propagation attributes.
func AddBranchPropagation(attributes map[string]schema.Attribute) {
	attributes["propagate_branches"] = schema.ListNestedAttribute{
		Optional: true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"propagate_to": schema.StringAttribute{
					Required: true,
					Validators: []validator.String{
						stringvalidator.LengthAtLeast(1),
						sovalidators.RegularExpression(),
					},
					MarkdownDescription: "Regular expression matching the branch names to propagate to.",
				},
			},
		},
		MarkdownDescription: "Rules for propagating assessments and observations to other branches.\n\n" +
			"~> Omit the attribute entirely to disable propagation. An empty list is **not** valid: SecObserve " +
			"stores it as null, which would leave Terraform with a permanent diff, so the provider rejects it " +
			"at plan time.",
	}
	attributes["propagate_branches_new_assessment"] = schema.BoolAttribute{
		Optional:            true,
		Computed:            true,
		Default:             booldefault.StaticBool(true),
		MarkdownDescription: "Propagate new assessments to the branches matched by `propagate_branches`.",
	}
	attributes["propagate_branches_new_observation"] = schema.BoolAttribute{
		Optional:            true,
		Computed:            true,
		Default:             booldefault.StaticBool(true),
		MarkdownDescription: "Propagate new observations to the branches matched by `propagate_branches`.",
	}
}

// ValidateBranchPropagation rejects an empty list, which SecObserve would
// normalize to null and thereby cause a permanent diff. Unknown is left
// alone: the eventual value isn't known yet, so there's nothing to validate
// until a later plan resolves it.
func (b BranchPropagation) ValidateBranchPropagation(diags *diag.Diagnostics) {
	if b.PropagateBranches.IsNull() || b.PropagateBranches.IsUnknown() {
		return
	}
	if len(b.PropagateBranches.Elements()) == 0 {
		diags.AddAttributeError(
			path.Root("propagate_branches"),
			"Empty propagate_branches list",
			"SecObserve stores an empty propagation list as null, so Terraform would report a diff on every "+
				"plan. Omit the attribute instead of setting it to [].",
		)
	}
}

// ToAPI converts the block into its request representation.
func (b BranchPropagation) ToAPI(ctx context.Context, diags *diag.Diagnostics) client.BranchPropagationFields {
	fields := client.BranchPropagationFields{
		PropagateBranchesNewAssessment:  tfutil.BoolValue(b.PropagateBranchesNewAssessment),
		PropagateBranchesNewObservation: tfutil.BoolValue(b.PropagateBranchesNewObservation),
	}
	if b.PropagateBranches.IsNull() || b.PropagateBranches.IsUnknown() {
		return fields
	}

	var rules []PropagateBranchModel
	diags.Append(b.PropagateBranches.ElementsAs(ctx, &rules, false)...)
	if diags.HasError() {
		return fields
	}
	for _, rule := range rules {
		fields.PropagateBranches = append(fields.PropagateBranches, client.PropagateBranch{
			PropagateTo: rule.PropagateTo.ValueString(),
		})
	}
	return fields
}

// FromAPI fills the block from an API response.
func (b *BranchPropagation) FromAPI(
	ctx context.Context, fields client.BranchPropagationFields, diags *diag.Diagnostics,
) {
	b.PropagateBranchesNewAssessment = types.BoolValue(fields.PropagateBranchesNewAssessment)
	b.PropagateBranchesNewObservation = types.BoolValue(fields.PropagateBranchesNewObservation)

	// Null and [] are the same thing here, and null is what SecObserve stores.
	if len(fields.PropagateBranches) == 0 {
		b.PropagateBranches = types.ListNull(PropagateBranchObjectType)
		return
	}

	rules := make([]PropagateBranchModel, 0, len(fields.PropagateBranches))
	for _, rule := range fields.PropagateBranches {
		rules = append(rules, PropagateBranchModel{PropagateTo: types.StringValue(rule.PropagateTo)})
	}
	list, d := types.ListValueFrom(ctx, PropagateBranchObjectType, rules)
	diags.Append(d...)
	b.PropagateBranches = list
}
