package schemacommon

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jabbrwcky/terraform-provider-secobserve/internal/client"
	"github.com/jabbrwcky/terraform-provider-secobserve/internal/tfutil"
	sovalidators "github.com/jabbrwcky/terraform-provider-secobserve/internal/validators"
)

// BranchPropagation is the branch propagation block of a product or product
// group.
type BranchPropagation struct {
	PropagateBranches               []PropagateBranchModel `tfsdk:"propagate_branches"`
	PropagateBranchesNewAssessment  types.Bool             `tfsdk:"propagate_branches_new_assessment"`
	PropagateBranchesNewObservation types.Bool             `tfsdk:"propagate_branches_new_observation"`
}

// PropagateBranchModel is one branch propagation rule.
type PropagateBranchModel struct {
	PropagateTo types.String `tfsdk:"propagate_to"`
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
// normalize to null and thereby cause a permanent diff.
func (b BranchPropagation) ValidateBranchPropagation(diags *diag.Diagnostics) {
	if b.PropagateBranches != nil && len(b.PropagateBranches) == 0 {
		diags.AddAttributeError(
			path.Root("propagate_branches"),
			"Empty propagate_branches list",
			"SecObserve stores an empty propagation list as null, so Terraform would report a diff on every "+
				"plan. Omit the attribute instead of setting it to [].",
		)
	}
}

// ToAPI converts the block into its request representation.
func (b BranchPropagation) ToAPI() client.BranchPropagationFields {
	fields := client.BranchPropagationFields{
		PropagateBranchesNewAssessment:  tfutil.BoolValue(b.PropagateBranchesNewAssessment),
		PropagateBranchesNewObservation: tfutil.BoolValue(b.PropagateBranchesNewObservation),
	}
	for _, rule := range b.PropagateBranches {
		fields.PropagateBranches = append(fields.PropagateBranches, client.PropagateBranch{
			PropagateTo: rule.PropagateTo.ValueString(),
		})
	}
	return fields
}

// FromAPI fills the block from an API response.
func (b *BranchPropagation) FromAPI(_ context.Context, fields client.BranchPropagationFields) {
	b.PropagateBranchesNewAssessment = types.BoolValue(fields.PropagateBranchesNewAssessment)
	b.PropagateBranchesNewObservation = types.BoolValue(fields.PropagateBranchesNewObservation)

	// Null and [] are the same thing here, and null is what SecObserve stores.
	if len(fields.PropagateBranches) == 0 {
		b.PropagateBranches = nil
		return
	}
	b.PropagateBranches = make([]PropagateBranchModel, 0, len(fields.PropagateBranches))
	for _, rule := range fields.PropagateBranches {
		b.PropagateBranches = append(b.PropagateBranches, PropagateBranchModel{
			PropagateTo: types.StringValue(rule.PropagateTo),
		})
	}
}
