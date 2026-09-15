package product_group

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/schemacommon"
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

// modelV0 mirrors the pre-v1 product_group model: identical to model, except
// security_gate/repository_branch_housekeeping are still the old flat blocks.
type modelV0 struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`

	securityGateV0
	branchHousekeepingV0
	schemacommon.Notifications
	schemacommon.Approvals
	schemacommon.Approvers
	schemacommon.RiskAcceptance
	schemacommon.LicensePolicy
	schemacommon.BranchPropagation
}

// securityGateV1/branchHousekeepingV1 mirror the shipped v1 model shape --
// SecurityGate/BranchHousekeeping as they existed between the
// SingleNestedAttribute change and the SingleNestedBlock change (`active`
// still a separate top-level bool, no `active` inside the nested object).
// Used only for decoding schema-version-1 state.
type securityGateV1 struct {
	SecurityGateActive types.Bool                `tfsdk:"security_gate_active"`
	Thresholds         *securityGateThresholdsV1 `tfsdk:"security_gate"`
}

type securityGateThresholdsV1 struct {
	Critical types.Int64 `tfsdk:"threshold_critical"`
	High     types.Int64 `tfsdk:"threshold_high"`
	Medium   types.Int64 `tfsdk:"threshold_medium"`
	Low      types.Int64 `tfsdk:"threshold_low"`
	None     types.Int64 `tfsdk:"threshold_none"`
	Unknown  types.Int64 `tfsdk:"threshold_unknown"`
}

type branchHousekeepingV1 struct {
	RepositoryBranchHousekeepingActive types.Bool                    `tfsdk:"repository_branch_housekeeping_active"`
	Settings                           *branchHousekeepingSettingsV1 `tfsdk:"repository_branch_housekeeping"`
}

type branchHousekeepingSettingsV1 struct {
	KeepInactiveDays types.Int64  `tfsdk:"keep_inactive_days"`
	ExemptBranches   types.String `tfsdk:"exempt_branches"`
}

// modelV1 mirrors the shipped v1 product_group model: identical to model,
// except security_gate/repository_branch_housekeeping are still the v1 shape.
type modelV1 struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`

	securityGateV1
	branchHousekeepingV1
	schemacommon.Notifications
	schemacommon.Approvals
	schemacommon.Approvers
	schemacommon.RiskAcceptance
	schemacommon.LicensePolicy
	schemacommon.BranchPropagation
}

// priorSchemaV1 reproduces the shipped v1 schema, identical to Schema()
// except for the two swapped blocks (still nested attributes at v1, not yet
// real blocks). Decode-only -- see schemacommon.AddSecurityGateV1.
func priorSchemaV1() *schema.Schema {
	attributes := map[string]schema.Attribute{
		"id": schema.Int64Attribute{
			Computed:      true,
			PlanModifiers: []planmodifier.Int64{schemacommon.Int64UseStateForUnknown()},
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
	}

	schemacommon.AddSecurityGateV1(attributes)
	schemacommon.AddBranchHousekeepingV1(attributes)
	schemacommon.AddNotifications(attributes)
	schemacommon.AddApprovals(attributes)
	schemacommon.AddApprovers(attributes, "product group")
	schemacommon.AddRiskAcceptance(attributes)
	schemacommon.AddLicensePolicy(attributes)
	schemacommon.AddBranchPropagation(attributes)

	return &schema.Schema{Version: 1, Attributes: attributes}
}

// priorSchemaV0 reproduces the pre-v1 schema, identical to Schema() except
// for the two swapped blocks. Decode-only -- see schemacommon.AddSecurityGateV0.
func priorSchemaV0() *schema.Schema {
	attributes := map[string]schema.Attribute{
		"id": schema.Int64Attribute{
			Computed:      true,
			PlanModifiers: []planmodifier.Int64{schemacommon.Int64UseStateForUnknown()},
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
	}

	schemacommon.AddSecurityGateV0(attributes)
	schemacommon.AddBranchHousekeepingV0(attributes)
	schemacommon.AddNotifications(attributes)
	schemacommon.AddApprovals(attributes)
	schemacommon.AddApprovers(attributes, "product group")
	schemacommon.AddRiskAcceptance(attributes)
	schemacommon.AddLicensePolicy(attributes)
	schemacommon.AddBranchPropagation(attributes)

	return &schema.Schema{Version: 0, Attributes: attributes}
}

func (r *productGroupResource) UpgradeState(context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema:   priorSchemaV0(),
			StateUpgrader: upgradeStateV0,
		},
		1: {
			PriorSchema:   priorSchemaV1(),
			StateUpgrader: upgradeStateV1,
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
	(&productGroupResource{}).Schema(ctx, resource.SchemaRequest{}, &currentSchema)
	resp.State = tfsdk.State{
		Schema: currentSchema.Schema,
		Raw:    tftypes.NewValue(currentSchema.Schema.Type().TerraformType(ctx), nil),
	}

	upgraded := modelFromV0(prior)
	resp.Diagnostics.Append(resp.State.Set(ctx, upgraded)...)
}

// upgradeStateV1 moves security_gate_active/repository_branch_housekeeping_active
// into the `active` field of the now-real security_gate/
// repository_branch_housekeeping blocks. Same one-time-diff caveat as
// upgradeStateV0.
func upgradeStateV1(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
	var prior modelV1
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var currentSchema resource.SchemaResponse
	(&productGroupResource{}).Schema(ctx, resource.SchemaRequest{}, &currentSchema)
	resp.State = tfsdk.State{
		Schema: currentSchema.Schema,
		Raw:    tftypes.NewValue(currentSchema.Schema.Type().TerraformType(ctx), nil),
	}

	upgraded := modelFromV1(prior)
	resp.Diagnostics.Append(resp.State.Set(ctx, upgraded)...)
}

func modelFromV1(prior modelV1) model {
	upgraded := model{
		ID:                prior.ID,
		Name:              prior.Name,
		Description:       prior.Description,
		Notifications:     prior.Notifications,
		Approvals:         prior.Approvals,
		Approvers:         prior.Approvers,
		RiskAcceptance:    prior.RiskAcceptance,
		LicensePolicy:     prior.LicensePolicy,
		BranchPropagation: prior.BranchPropagation,
	}

	upgraded.SecurityGate.Block = securityGateBlockFromV1(prior.securityGateV1)
	upgraded.BranchHousekeeping.Block = housekeepingBlockFromV1(prior.branchHousekeepingV1)

	return upgraded
}

// securityGateBlockFromV1 folds the v1 top-level security_gate_active and
// the v1 nested thresholds-only object into the new block's active field
// plus thresholds. No block at all (inherit) only when both are absent.
func securityGateBlockFromV1(v1 securityGateV1) *schemacommon.SecurityGateBlock {
	if v1.SecurityGateActive.IsNull() && v1.Thresholds == nil {
		return nil
	}
	block := &schemacommon.SecurityGateBlock{Active: v1.SecurityGateActive}
	if v1.Thresholds != nil {
		block.Critical = v1.Thresholds.Critical
		block.High = v1.Thresholds.High
		block.Medium = v1.Thresholds.Medium
		block.Low = v1.Thresholds.Low
		block.None = v1.Thresholds.None
		block.Unknown = v1.Thresholds.Unknown
	}
	return block
}

// housekeepingBlockFromV1 is the housekeeping equivalent of
// securityGateBlockFromV1.
func housekeepingBlockFromV1(v1 branchHousekeepingV1) *schemacommon.BranchHousekeepingBlock {
	if v1.RepositoryBranchHousekeepingActive.IsNull() && v1.Settings == nil {
		return nil
	}
	block := &schemacommon.BranchHousekeepingBlock{Active: v1.RepositoryBranchHousekeepingActive}
	if v1.Settings != nil {
		block.KeepInactiveDays = v1.Settings.KeepInactiveDays
		block.ExemptBranches = v1.Settings.ExemptBranches
	}
	return block
}

func modelFromV0(prior modelV0) model {
	upgraded := model{
		ID:                prior.ID,
		Name:              prior.Name,
		Description:       prior.Description,
		Notifications:     prior.Notifications,
		Approvals:         prior.Approvals,
		Approvers:         prior.Approvers,
		RiskAcceptance:    prior.RiskAcceptance,
		LicensePolicy:     prior.LicensePolicy,
		BranchPropagation: prior.BranchPropagation,
	}

	upgraded.SecurityGate.Block = securityGateBlockFromV0(prior.securityGateV0)
	upgraded.BranchHousekeeping.Block = housekeepingBlockFromV0(prior.branchHousekeepingV0)

	return upgraded
}

// securityGateBlockFromV0 folds the old top-level security_gate_active into
// the new block's active field. A null v0 active (inherit) with every
// threshold also null becomes no block at all (inherit); anything else
// becomes a block, active carried across exactly.
func securityGateBlockFromV0(v0 securityGateV0) *schemacommon.SecurityGateBlock {
	if v0.SecurityGateActive.IsNull() &&
		v0.SecurityGateThresholdCritical.IsNull() && v0.SecurityGateThresholdHigh.IsNull() &&
		v0.SecurityGateThresholdMedium.IsNull() && v0.SecurityGateThresholdLow.IsNull() &&
		v0.SecurityGateThresholdNone.IsNull() && v0.SecurityGateThresholdUnknown.IsNull() {
		return nil
	}
	return &schemacommon.SecurityGateBlock{
		Active:   v0.SecurityGateActive,
		Critical: v0.SecurityGateThresholdCritical,
		High:     v0.SecurityGateThresholdHigh,
		Medium:   v0.SecurityGateThresholdMedium,
		Low:      v0.SecurityGateThresholdLow,
		None:     v0.SecurityGateThresholdNone,
		Unknown:  v0.SecurityGateThresholdUnknown,
	}
}

// housekeepingBlockFromV0 is the housekeeping equivalent of
// securityGateBlockFromV0.
func housekeepingBlockFromV0(v0 branchHousekeepingV0) *schemacommon.BranchHousekeepingBlock {
	exempt := v0.RepositoryBranchHousekeepingExemptBranches
	if v0.RepositoryBranchHousekeepingActive.IsNull() &&
		v0.RepositoryBranchHousekeepingKeepInactiveDays.IsNull() && (exempt.IsNull() || exempt.ValueString() == "") {
		return nil
	}
	return &schemacommon.BranchHousekeepingBlock{
		Active:           v0.RepositoryBranchHousekeepingActive,
		KeepInactiveDays: v0.RepositoryBranchHousekeepingKeepInactiveDays,
		ExemptBranches:   exempt,
	}
}
