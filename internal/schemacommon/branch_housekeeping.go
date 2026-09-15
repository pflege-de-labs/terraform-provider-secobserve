package schemacommon

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tfutil"
	sovalidators "github.com/pflege-de-labs/terraform-provider-secobserve/internal/validators"
)

// BranchHousekeeping is the branch housekeeping block of a product or product
// group.
type BranchHousekeeping struct {
	RepositoryBranchHousekeepingActive           types.Bool   `tfsdk:"repository_branch_housekeeping_active"`
	RepositoryBranchHousekeepingKeepInactiveDays types.Int64  `tfsdk:"repository_branch_housekeeping_keep_inactive_days"`
	RepositoryBranchHousekeepingExemptBranches   types.String `tfsdk:"repository_branch_housekeeping_exempt_branches"`
}

// AddBranchHousekeeping contributes the branch housekeeping attributes.
func AddBranchHousekeeping(attributes map[string]schema.Attribute) {
	attributes["repository_branch_housekeeping_active"] = schema.BoolAttribute{
		Optional: true,
		MarkdownDescription: "Whether inactive branches are deleted automatically.\n\n" +
			"Tri-state: leave it unset to inherit from the product group or, failing that, from the instance " +
			"settings. That is different from `false`, which disables housekeeping explicitly.\n\n" +
			"Setting it to `false` also clears `repository_branch_housekeeping_keep_inactive_days` and " +
			"`repository_branch_housekeeping_exempt_branches` server-side -- both are rejected at plan time " +
			"if set explicitly alongside `false`.",
	}
	attributes["repository_branch_housekeeping_keep_inactive_days"] = schema.Int64Attribute{
		Optional:      true,
		Computed:      true,
		Validators:    []validator.Int64{int64validator.Between(1, 999999)},
		PlanModifiers: []planmodifier.Int64{Int64UnknownWhenBoolSiblingChanges("repository_branch_housekeeping_active")},
		MarkdownDescription: "Days a branch may stay inactive before housekeeping deletes it. Leave it unset " +
			"to inherit the instance-wide default.\n\n" +
			"~> Cannot be set while `repository_branch_housekeeping_active` is explicitly `false`.",
	}
	attributes["repository_branch_housekeeping_exempt_branches"] = schema.StringAttribute{
		Optional:      true,
		Computed:      true,
		Default:       stringdefault.StaticString(""),
		PlanModifiers: []planmodifier.String{StringUnknownWhenBoolSiblingChanges("repository_branch_housekeeping_active")},
		Validators: []validator.String{
			stringvalidator.LengthAtMost(255),
			sovalidators.RegularExpression(),
		},
		MarkdownDescription: "Regular expression matching branch names that housekeeping must never delete.\n\n" +
			"~> Cannot be set while `repository_branch_housekeeping_active` is explicitly `false`.",
	}
}

// ToAPI converts the block into its request representation.
func (b BranchHousekeeping) ToAPI() client.BranchHousekeepingFields {
	return client.BranchHousekeepingFields{
		RepositoryBranchHousekeepingActive:           tfutil.BoolPtr(b.RepositoryBranchHousekeepingActive),
		RepositoryBranchHousekeepingKeepInactiveDays: tfutil.Int64Ptr(b.RepositoryBranchHousekeepingKeepInactiveDays),
		RepositoryBranchHousekeepingExemptBranches:   tfutil.StringValue(b.RepositoryBranchHousekeepingExemptBranches),
	}
}

// FromAPI fills the block from an API response.
func (b *BranchHousekeeping) FromAPI(fields client.BranchHousekeepingFields) {
	b.RepositoryBranchHousekeepingActive = tfutil.Bool(fields.RepositoryBranchHousekeepingActive)
	b.RepositoryBranchHousekeepingKeepInactiveDays = tfutil.Int64(fields.RepositoryBranchHousekeepingKeepInactiveDays)
	b.RepositoryBranchHousekeepingExemptBranches = types.StringValue(fields.RepositoryBranchHousekeepingExemptBranches)
}

// ValidateBranchHousekeeping rejects an explicit keep_inactive_days or
// exempt_branches value combined with repository_branch_housekeeping_active
// explicitly set to false.
//
// SecObserve force-clears both fields server-side whenever active is false
// (core/api/serializers_product.py:112-120), regardless of what was
// submitted. Terraform forbids an applied value that differs from the planned
// one for an attribute the practitioner wrote, so honouring an explicit value
// here would fail the apply with "provider produced inconsistent result".
// Rejecting the combination at plan time is both earlier and explicable.
func (b BranchHousekeeping) ValidateBranchHousekeeping(diags *diag.Diagnostics) {
	active := b.RepositoryBranchHousekeepingActive
	if active.IsUnknown() || active.IsNull() || active.ValueBool() {
		// Only an explicit, known false triggers SecObserve's unconditional
		// clear; null (inherit) and true leave both fields alone.
		return
	}

	if !b.RepositoryBranchHousekeepingKeepInactiveDays.IsNull() && !b.RepositoryBranchHousekeepingKeepInactiveDays.IsUnknown() {
		diags.AddAttributeError(
			path.Root("repository_branch_housekeeping_keep_inactive_days"),
			"Cannot be set while housekeeping is inactive",
			"SecObserve clears repository_branch_housekeeping_keep_inactive_days server-side whenever "+
				"repository_branch_housekeeping_active is false, regardless of what is configured here.\n\n"+
				"Remove this attribute, or set repository_branch_housekeeping_active to true or leave it unset.",
		)
	}

	if value := b.RepositoryBranchHousekeepingExemptBranches; !value.IsNull() && !value.IsUnknown() && value.ValueString() != "" {
		diags.AddAttributeError(
			path.Root("repository_branch_housekeeping_exempt_branches"),
			"Cannot be set while housekeeping is inactive",
			"SecObserve clears repository_branch_housekeeping_exempt_branches server-side whenever "+
				"repository_branch_housekeeping_active is false, regardless of what is configured here.\n\n"+
				"Remove this attribute, or set repository_branch_housekeeping_active to true or leave it unset.",
		)
	}
}
