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

// BranchHousekeeping is the repository_branch_housekeeping block of a product
// or product group.
type BranchHousekeeping struct {
	Block *BranchHousekeepingBlock `tfsdk:"repository_branch_housekeeping"`
}

// BranchHousekeepingBlock is the nested repository_branch_housekeeping
// block. Absent entirely means inherit from the product group or the
// instance settings (tri-state); a present block with no `active` means
// active -- setting keep_inactive_days/exempt_branches implies activating
// housekeeping. `active = false` inside the block is the explicit way to
// switch it off.
//
// The block deliberately carries no server-filled values: state is built
// from the plan, never from the API response -- see the package doc comment
// on plan_modifiers.go.
type BranchHousekeepingBlock struct {
	Active           types.Bool   `tfsdk:"active"`
	KeepInactiveDays types.Int64  `tfsdk:"keep_inactive_days"`
	ExemptBranches   types.String `tfsdk:"exempt_branches"`
}

// AddBranchHousekeeping contributes the repository_branch_housekeeping block.
func AddBranchHousekeeping(blocks map[string]schema.Block) {
	blocks["repository_branch_housekeeping"] = schema.SingleNestedBlock{
		Attributes: map[string]schema.Attribute{
			"active": schema.BoolAttribute{
				Optional: true,
				MarkdownDescription: "Explicitly switches housekeeping off when set to `false`. Leave unset " +
					"(or `true`) to activate housekeeping -- the block's mere presence already does that, this " +
					"exists so the block can also express \"explicitly off\" without being removed.\n\n" +
					"~> If this product belongs to a product group, an explicit `true`/`false` on the " +
					"**product group** always wins over this one -- this product's own value only applies when " +
					"the product group's is left unset. See `docs/design/api-quirks.md` for the source " +
					"reference.",
			},
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
		MarkdownDescription: "Branch housekeeping configuration. Omit this block entirely to inherit from " +
			"the product group or, failing that, from the instance settings (which defaults to active). " +
			"Writing the block -- even empty -- activates housekeeping; set `active = false` inside it to " +
			"switch housekeeping off explicitly instead of inheriting.\n\n" +
			"SecObserve clears both other fields server-side whenever housekeeping ends up inactive, so " +
			"either is rejected at plan time if set alongside `active = false`.\n\n" +
			"Leave a field unset to inherit the instance-wide default -- the default is not reflected back " +
			"into this block, matching the rest of this provider's read-only/informational data being " +
			"reserved for data sources.\n\n" +
			"~> Unlike every other attribute in this provider, this block's state is echoed from your " +
			"configuration rather than read back from SecObserve. Two consequences: `terraform import` cannot " +
			"recover a configured housekeeping setup (add the block to your configuration afterwards to match " +
			"what's actually configured), and changing it directly in SecObserve rather than through " +
			"Terraform will not be detected as drift.\n\n" +
			"~> If this attribute previously tracked values it no longer does -- narrowing the block, or " +
			"upgrading from a provider version where these fields were server-computed -- the next plan clears " +
			"them from state. This is safe and happens once: housekeeping stays active and SecObserve still " +
			"fills unset fields with its own defaults server-side, only Terraform's state stops tracking " +
			"values your configuration never asked for. The plan is empty again immediately after.",
	}
}

// ToAPI converts the block into its request representation. A present block
// with `active` unset resolves to active = true; keep_inactive_days/
// exempt_branches are sent as configured regardless, since
// ValidateBranchHousekeeping is what rejects them alongside an explicit
// active = false.
func (b BranchHousekeeping) ToAPI() client.BranchHousekeepingFields {
	if b.Block == nil {
		return client.BranchHousekeepingFields{}
	}

	active := true
	if !b.Block.Active.IsNull() && !b.Block.Active.IsUnknown() {
		active = b.Block.Active.ValueBool()
	}

	return client.BranchHousekeepingFields{
		RepositoryBranchHousekeepingActive:           &active,
		RepositoryBranchHousekeepingKeepInactiveDays: tfutil.Int64Ptr(b.Block.KeepInactiveDays),
		RepositoryBranchHousekeepingExemptBranches:   tfutil.StringValue(b.Block.ExemptBranches),
	}
}

// FromAPI intentionally does nothing: repository_branch_housekeeping's state
// is carried forward from the plan/prior state by the resource's
// Create/Read/Update, not read back from the response. See the package doc
// comment on plan_modifiers.go.
func (b *BranchHousekeeping) FromAPI(client.BranchHousekeepingFields) {}

// ValidateBranchHousekeeping rejects keep_inactive_days/exempt_branches set
// alongside an explicit active = false.
//
// SecObserve force-clears both fields server-side whenever housekeeping is
// inactive (core/api/serializers_product.py:112-120), regardless of what was
// submitted. Terraform forbids an applied value that differs from the planned
// one for an attribute the practitioner wrote, so honouring an explicit value
// here would fail the apply with "provider produced inconsistent result".
// Rejecting the combination at plan time is both earlier and explicable.
func (b BranchHousekeeping) ValidateBranchHousekeeping(diags *diag.Diagnostics) {
	if b.Block == nil {
		return
	}

	active := b.Block.Active
	if active.IsUnknown() || active.IsNull() || active.ValueBool() {
		// Block present with active unset or true: housekeeping is on, both
		// fields are meaningful.
		return
	}

	if !b.Block.KeepInactiveDays.IsNull() && !b.Block.KeepInactiveDays.IsUnknown() {
		diags.AddAttributeError(
			path.Root("repository_branch_housekeeping").AtName("keep_inactive_days"),
			"Cannot be set while housekeeping is inactive",
			"SecObserve clears keep_inactive_days server-side whenever housekeeping is inactive, "+
				"regardless of what is configured here.\n\n"+
				"Remove this attribute, or remove active = false from the repository_branch_housekeeping block.",
		)
	}

	if value := b.Block.ExemptBranches; !value.IsNull() && !value.IsUnknown() && value.ValueString() != "" {
		diags.AddAttributeError(
			path.Root("repository_branch_housekeeping").AtName("exempt_branches"),
			"Cannot be set while housekeeping is inactive",
			"SecObserve clears exempt_branches server-side whenever housekeeping is inactive, "+
				"regardless of what is configured here.\n\n"+
				"Remove this attribute, or remove active = false from the repository_branch_housekeeping block.",
		)
	}
}
