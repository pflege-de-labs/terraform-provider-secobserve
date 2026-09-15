package schemacommon

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tfutil"
	sovalidators "github.com/pflege-de-labs/terraform-provider-secobserve/internal/validators"
)

// BranchHousekeeping is the branch housekeeping block of a product or product
// group.
type BranchHousekeeping struct {
	RepositoryBranchHousekeepingActive types.Bool                  `tfsdk:"repository_branch_housekeeping_active"`
	Settings                           *BranchHousekeepingSettings `tfsdk:"repository_branch_housekeeping"`
}

// BranchHousekeepingSettings is the nested repository_branch_housekeeping
// object. Like SecurityGateThresholds, it carries no server-filled values:
// state is built from the plan, never from the API response -- see the
// package doc comment on plan_modifiers.go.
type BranchHousekeepingSettings struct {
	KeepInactiveDays types.Int64  `tfsdk:"keep_inactive_days"`
	ExemptBranches   types.String `tfsdk:"exempt_branches"`
}

// AddBranchHousekeeping contributes the branch housekeeping attributes.
func AddBranchHousekeeping(attributes map[string]schema.Attribute) {
	attributes["repository_branch_housekeeping_active"] = schema.BoolAttribute{
		Optional: true,
		MarkdownDescription: "Whether inactive branches are deleted automatically.\n\n" +
			"Tri-state: leave it unset to inherit from the product group or, failing that, from the instance " +
			"settings (which defaults to `true`). That is different from `false`, which disables housekeeping " +
			"explicitly.\n\n" +
			"~> If this product belongs to a product group, an explicit `true`/`false` on the **product group** " +
			"always wins over this attribute -- this product's own value only applies when the product group's " +
			"is left unset. See `docs/design/api-quirks.md` for the source reference.\n\n" +
			"Setting it to `false` also clears `repository_branch_housekeeping` server-side -- the block is " +
			"rejected at plan time if set alongside `false`.",
	}
	attributes["repository_branch_housekeeping"] = schema.SingleNestedAttribute{
		Optional: true,
		Attributes: map[string]schema.Attribute{
			"keep_inactive_days": schema.Int64Attribute{
				Optional:            true,
				Validators:          []validator.Int64{int64validator.Between(1, 999999)},
				MarkdownDescription: "Days a branch may stay inactive before housekeeping deletes it.",
			},
			"exempt_branches": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(255),
					sovalidators.RegularExpression(),
				},
				MarkdownDescription: "Regular expression matching branch names that housekeeping must never delete.",
			},
		},
		MarkdownDescription: "Branch housekeeping settings. Only meaningful while " +
			"`repository_branch_housekeeping_active` is `true`; SecObserve clears both fields server-side " +
			"otherwise, and this attribute is rejected at plan time if set alongside " +
			"`repository_branch_housekeeping_active = false`.\n\n" +
			"Leave a field unset to inherit the instance-wide default -- the default is not reflected back " +
			"into this block, matching the rest of this provider's read-only/informational data being " +
			"reserved for data sources.\n\n" +
			"~> Unlike every other attribute in this provider, this block's state is echoed from your " +
			"configuration rather than read back from SecObserve. Two consequences: `terraform import` cannot " +
			"recover configured settings (add them to your configuration afterwards to match what's actually " +
			"configured), and changing a setting directly in SecObserve rather than through Terraform will not " +
			"be detected as drift.",
	}
}

// ToAPI converts the block into its request representation.
func (b BranchHousekeeping) ToAPI() client.BranchHousekeepingFields {
	fields := client.BranchHousekeepingFields{
		RepositoryBranchHousekeepingActive: tfutil.BoolPtr(b.RepositoryBranchHousekeepingActive),
	}
	if b.Settings != nil {
		fields.RepositoryBranchHousekeepingKeepInactiveDays = tfutil.Int64Ptr(b.Settings.KeepInactiveDays)
		fields.RepositoryBranchHousekeepingExemptBranches = tfutil.StringValue(b.Settings.ExemptBranches)
	}
	return fields
}

// FromAPI fills the block from an API response.
//
// Deliberately does NOT populate Settings: unlike the rest of this provider,
// repository_branch_housekeeping's state is carried forward from the
// plan/prior state by the resource's Create/Read/Update, not read back from
// the response. See the package doc comment on plan_modifiers.go.
func (b *BranchHousekeeping) FromAPI(fields client.BranchHousekeepingFields) {
	b.RepositoryBranchHousekeepingActive = tfutil.Bool(fields.RepositoryBranchHousekeepingActive)
}

// ValidateBranchHousekeeping rejects a repository_branch_housekeeping block
// combined with repository_branch_housekeeping_active explicitly set to
// false, and warns when the block is set but active is left unset.
//
// SecObserve force-clears both fields server-side whenever active is false
// (core/api/serializers_product.py:112-120), regardless of what was
// submitted. Terraform forbids an applied value that differs from the planned
// one for an attribute the practitioner wrote, so honouring an explicit value
// here would fail the apply with "provider produced inconsistent result".
// Rejecting the combination at plan time is both earlier and explicable.
func (b BranchHousekeeping) ValidateBranchHousekeeping(diags *diag.Diagnostics) {
	if b.Settings == nil {
		return
	}

	active := b.RepositoryBranchHousekeepingActive
	if active.IsUnknown() {
		return
	}

	if !active.IsNull() && !active.ValueBool() {
		diags.AddAttributeError(
			path.Root("repository_branch_housekeeping"),
			"Cannot be set while housekeeping is inactive",
			"SecObserve clears repository_branch_housekeeping server-side whenever "+
				"repository_branch_housekeeping_active is false, regardless of what is configured here.\n\n"+
				"Remove this block, or set repository_branch_housekeeping_active to true.",
		)
		return
	}

	if active.IsNull() {
		diags.AddAttributeWarning(
			path.Root("repository_branch_housekeeping"),
			"Ignored unless repository_branch_housekeeping_active is true",
			"repository_branch_housekeeping_active is unset here, so SecObserve never consults these "+
				"settings on this resource -- whatever ends up active comes from the product group or the "+
				"instance-wide default instead, using their own settings. Set "+
				"repository_branch_housekeeping_active = true to make this resource's settings apply.",
		)
	}
}
