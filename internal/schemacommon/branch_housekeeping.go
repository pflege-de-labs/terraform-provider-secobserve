package schemacommon

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jabbrwcky/terraform-provider-secobserve/internal/client"
	"github.com/jabbrwcky/terraform-provider-secobserve/internal/tfutil"
	sovalidators "github.com/jabbrwcky/terraform-provider-secobserve/internal/validators"
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
			"`repository_branch_housekeeping_exempt_branches` server-side.",
	}
	attributes["repository_branch_housekeeping_keep_inactive_days"] = schema.Int64Attribute{
		Optional:      true,
		Computed:      true,
		Validators:    []validator.Int64{int64validator.Between(1, 999999)},
		PlanModifiers: []planmodifier.Int64{Int64UnknownWhenBoolSiblingChanges("repository_branch_housekeeping_active")},
		MarkdownDescription: "Days a branch may stay inactive before housekeeping deletes it. Leave it unset " +
			"to inherit the instance-wide default.",
	}
	attributes["repository_branch_housekeeping_exempt_branches"] = schema.StringAttribute{
		Optional: true,
		Computed: true,
		Default:  stringdefault.StaticString(""),
		Validators: []validator.String{
			stringvalidator.LengthAtMost(255),
			sovalidators.RegularExpression(),
		},
		MarkdownDescription: "Regular expression matching branch names that housekeeping must never delete.",
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
