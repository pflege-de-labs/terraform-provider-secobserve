package product

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/schemacommon"
	sovalidators "github.com/pflege-de-labs/terraform-provider-secobserve/internal/validators"
)

// securityGateV0/branchHousekeepingV0 mirror the pre-v1 flat model shape --
// the exact struct definitions schemacommon.SecurityGate/BranchHousekeeping
// had before the security_gate/repository_branch_housekeeping nested
// attributes replaced them. Used only for decoding schema-version-0 state.
type securityGateV0 struct {
	SecurityGateActive            types.Bool  `tfsdk:"security_gate_active"`
	SecurityGateThresholdCritical types.Int64 `tfsdk:"security_gate_threshold_critical"`
	SecurityGateThresholdHigh     types.Int64 `tfsdk:"security_gate_threshold_high"`
	SecurityGateThresholdMedium   types.Int64 `tfsdk:"security_gate_threshold_medium"`
	SecurityGateThresholdLow      types.Int64 `tfsdk:"security_gate_threshold_low"`
	SecurityGateThresholdNone     types.Int64 `tfsdk:"security_gate_threshold_none"`
	SecurityGateThresholdUnknown  types.Int64 `tfsdk:"security_gate_threshold_unknown"`
}

type branchHousekeepingV0 struct {
	RepositoryBranchHousekeepingActive           types.Bool   `tfsdk:"repository_branch_housekeeping_active"`
	RepositoryBranchHousekeepingKeepInactiveDays types.Int64  `tfsdk:"repository_branch_housekeeping_keep_inactive_days"`
	RepositoryBranchHousekeepingExemptBranches   types.String `tfsdk:"repository_branch_housekeeping_exempt_branches"`
}

// modelV0 mirrors the pre-v1 product model: identical to model, except
// security_gate/repository_branch_housekeeping are still the old flat blocks.
type modelV0 struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`

	ProductGroup     types.Int64  `tfsdk:"product_group"`
	ProductGroupName types.String `tfsdk:"product_group_name"`

	RepositoryPrefix            types.String `tfsdk:"repository_prefix"`
	RepositoryDefaultBranch     types.Int64  `tfsdk:"repository_default_branch"`
	RepositoryDefaultBranchName types.String `tfsdk:"repository_default_branch_name"`

	Purl  types.String `tfsdk:"purl"`
	CPE23 types.String `tfsdk:"cpe23"`

	ApplyGeneralRules  types.Bool `tfsdk:"apply_general_rules"`
	SecurityGatePassed types.Bool `tfsdk:"security_gate_passed"`

	securityGateV0
	branchHousekeepingV0
	schemacommon.Notifications
	schemacommon.Approvals
	schemacommon.Approvers
	schemacommon.RiskAcceptance
	schemacommon.LicensePolicy
	schemacommon.BranchPropagation

	issueTracker
	scanners
}

// priorSchemaV0 reproduces the pre-v1 schema, identical to Schema() except
// for the two swapped blocks. Decode-only -- see schemacommon.AddSecurityGateV0.
func priorSchemaV0() *schema.Schema {
	attributes := map[string]schema.Attribute{
		"id": schema.Int64Attribute{
			Computed:            true,
			PlanModifiers:       []planmodifier.Int64{schemacommon.Int64UseStateForUnknown()},
			MarkdownDescription: "Numeric id of the product.",
		},
		"name": schema.StringAttribute{
			Required:   true,
			Validators: []validator.String{stringvalidator.LengthBetween(1, 255)},
		},
		"description": schema.StringAttribute{
			Optional: true,
			Computed: true,
			Default:  stringdefault.StaticString(""),
		},
		"product_group": schema.Int64Attribute{
			Optional:   true,
			Validators: []validator.Int64{int64validator.AtLeast(1)},
		},
		"product_group_name": schema.StringAttribute{
			Computed: true,
		},
		"repository_prefix": schema.StringAttribute{
			Optional: true,
			Computed: true,
			Default:  stringdefault.StaticString(""),
			Validators: []validator.String{
				stringvalidator.LengthAtMost(255),
				sovalidators.HTTPURL(),
			},
		},
		"repository_default_branch": schema.Int64Attribute{
			Computed: true,
		},
		"repository_default_branch_name": schema.StringAttribute{
			Computed: true,
		},
		"purl": schema.StringAttribute{
			Optional: true,
			Computed: true,
			Default:  stringdefault.StaticString(""),
		},
		"cpe23": schema.StringAttribute{
			Optional: true,
			Computed: true,
			Default:  stringdefault.StaticString(""),
		},
		"apply_general_rules": schema.BoolAttribute{
			Optional: true,
			Computed: true,
			Default:  booldefault.StaticBool(true),
		},
		"security_gate_passed": schema.BoolAttribute{
			Computed: true,
		},
	}

	schemacommon.AddSecurityGateV0(attributes)
	schemacommon.AddBranchHousekeepingV0(attributes)
	schemacommon.AddNotifications(attributes)
	schemacommon.AddApprovals(attributes)
	schemacommon.AddApprovers(attributes, "product or its product group")
	schemacommon.AddRiskAcceptance(attributes)
	schemacommon.AddLicensePolicy(attributes)
	schemacommon.AddBranchPropagation(attributes)
	addIssueTracker(attributes)
	addScanners(attributes)

	return &schema.Schema{Version: 0, Attributes: attributes}
}

func (r *productResource) UpgradeState(context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema:   priorSchemaV0(),
			StateUpgrader: upgradeStateV0,
		},
	}
}

// upgradeStateV0 moves the flat security_gate_threshold_*/
// repository_branch_housekeeping_{keep_inactive_days,exempt_branches} values
// into the new nested blocks, carrying every non-null value forward
// unconditionally. Prior state cannot distinguish a practitioner-configured
// value from one SecObserve filled in server-side, so this may produce a
// one-time plan diff clearing values that were never actually in the
// practitioner's config -- expected, and resolved by a single apply.
//
// Existing .tf files still need to be hand-edited to the new nested syntax;
// this upgrader only prevents a "terraform state rm" + reimport, it doesn't
// let old flat HCL keep working.
func upgradeStateV0(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
	var prior modelV0
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// resp.State starts as the zero value; Set needs a schema to reflect
	// against, or it never terminates.
	var currentSchema resource.SchemaResponse
	(&productResource{}).Schema(ctx, resource.SchemaRequest{}, &currentSchema)
	resp.State = tfsdk.State{
		Schema: currentSchema.Schema,
		Raw:    tftypes.NewValue(currentSchema.Schema.Type().TerraformType(ctx), nil),
	}

	upgraded := modelFromV0(prior)
	resp.Diagnostics.Append(resp.State.Set(ctx, upgraded)...)
}

func modelFromV0(prior modelV0) model {
	upgraded := model{
		ID:                          prior.ID,
		Name:                        prior.Name,
		Description:                 prior.Description,
		ProductGroup:                prior.ProductGroup,
		ProductGroupName:            prior.ProductGroupName,
		RepositoryPrefix:            prior.RepositoryPrefix,
		RepositoryDefaultBranch:     prior.RepositoryDefaultBranch,
		RepositoryDefaultBranchName: prior.RepositoryDefaultBranchName,
		Purl:                        prior.Purl,
		CPE23:                       prior.CPE23,
		ApplyGeneralRules:           prior.ApplyGeneralRules,
		SecurityGatePassed:          prior.SecurityGatePassed,
		Notifications:               prior.Notifications,
		Approvals:                   prior.Approvals,
		Approvers:                   prior.Approvers,
		RiskAcceptance:              prior.RiskAcceptance,
		LicensePolicy:               prior.LicensePolicy,
		BranchPropagation:           prior.BranchPropagation,
		issueTracker:                prior.issueTracker,
		scanners:                    prior.scanners,
	}

	upgraded.SecurityGate.SecurityGateActive = prior.SecurityGateActive
	upgraded.SecurityGate.Thresholds = thresholdsFromV0(prior.securityGateV0)

	upgraded.BranchHousekeeping.RepositoryBranchHousekeepingActive = prior.RepositoryBranchHousekeepingActive
	upgraded.BranchHousekeeping.Settings = housekeepingSettingsFromV0(prior.branchHousekeepingV0)

	return upgraded
}

func thresholdsFromV0(v0 securityGateV0) *schemacommon.SecurityGateThresholds {
	if v0.SecurityGateThresholdCritical.IsNull() && v0.SecurityGateThresholdHigh.IsNull() &&
		v0.SecurityGateThresholdMedium.IsNull() && v0.SecurityGateThresholdLow.IsNull() &&
		v0.SecurityGateThresholdNone.IsNull() && v0.SecurityGateThresholdUnknown.IsNull() {
		return nil
	}
	return &schemacommon.SecurityGateThresholds{
		Critical: v0.SecurityGateThresholdCritical,
		High:     v0.SecurityGateThresholdHigh,
		Medium:   v0.SecurityGateThresholdMedium,
		Low:      v0.SecurityGateThresholdLow,
		None:     v0.SecurityGateThresholdNone,
		Unknown:  v0.SecurityGateThresholdUnknown,
	}
}

func housekeepingSettingsFromV0(v0 branchHousekeepingV0) *schemacommon.BranchHousekeepingSettings {
	exempt := v0.RepositoryBranchHousekeepingExemptBranches
	if v0.RepositoryBranchHousekeepingKeepInactiveDays.IsNull() && (exempt.IsNull() || exempt.ValueString() == "") {
		return nil
	}
	return &schemacommon.BranchHousekeepingSettings{
		KeepInactiveDays: v0.RepositoryBranchHousekeepingKeepInactiveDays,
		ExemptBranches:   exempt,
	}
}
