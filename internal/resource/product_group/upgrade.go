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

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
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
			StateUpgrader: r.upgradeStateV0,
		},
		1: {
			PriorSchema:   priorSchemaV1(),
			StateUpgrader: upgradeStateV1,
		},
	}
}

// upgradeStateV0 moves the flat security_gate_threshold_*/
// repository_branch_housekeeping_{keep_inactive_days,exempt_branches} values
// into the new blocks. At v0 those fields were Optional+Computed, so a
// non-null value in prior state could be either a practitioner-configured
// value or one SecObserve filled in server-side -- unlike everything else in
// this provider, prior state alone can't tell the two apart. Fetching the
// current instance-wide defaults and dropping any value that matches one
// resolves the ambiguity safely: a value equal to the default produces the
// identical wire request whether it's kept or dropped (SecObserve fills the
// same default either way), so dropping it can never lose real configuration
// -- at worst, a value that coincidentally equals the default gets dropped
// and then re-added by the next apply if the practitioner's actual .tf still
// has it, which is a harmless one-line diff, not data loss.
//
// Existing .tf files still need to be hand-edited to the new block syntax;
// this upgrader only prevents a "terraform state rm" + reimport, it doesn't
// let old flat HCL keep working.
func (r *productGroupResource) upgradeStateV0(
	ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse,
) {
	var prior modelV0
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	settings, err := r.client.GetSettings(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Could not read SecObserve instance settings while upgrading state",
			"Upgrading from the pre-v1 schema needs the current security gate/branch housekeeping "+
				"defaults to tell a practitioner-configured threshold apart from one SecObserve filled in "+
				"server-side.\n\nUnderlying error: "+err.Error(),
		)
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

	upgraded := modelFromV0(prior, settings)
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

func modelFromV0(prior modelV0, settings client.Settings) model {
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

	upgraded.SecurityGate.Block = securityGateBlockFromV0(prior.securityGateV0, settings)
	upgraded.BranchHousekeeping.Block = housekeepingBlockFromV0(prior.branchHousekeepingV0, settings)

	return upgraded
}

// securityGateBlockFromV0 folds the old top-level security_gate_active into
// the new block's active field, dropping any threshold that matches the
// current instance-wide default (see upgradeStateV0's doc comment for why
// that's safe). A null v0 active (inherit) with every threshold also
// null-or-dropped becomes no block at all (inherit).
func securityGateBlockFromV0(v0 securityGateV0, settings client.Settings) *schemacommon.SecurityGateBlock {
	critical := dropIfDefaultInt64(v0.SecurityGateThresholdCritical, settings.SecurityGateThresholdCritical)
	high := dropIfDefaultInt64(v0.SecurityGateThresholdHigh, settings.SecurityGateThresholdHigh)
	medium := dropIfDefaultInt64(v0.SecurityGateThresholdMedium, settings.SecurityGateThresholdMedium)
	low := dropIfDefaultInt64(v0.SecurityGateThresholdLow, settings.SecurityGateThresholdLow)
	none := dropIfDefaultInt64(v0.SecurityGateThresholdNone, settings.SecurityGateThresholdNone)
	unknown := dropIfDefaultInt64(v0.SecurityGateThresholdUnknown, settings.SecurityGateThresholdUnknown)

	if v0.SecurityGateActive.IsNull() &&
		critical.IsNull() && high.IsNull() && medium.IsNull() && low.IsNull() && none.IsNull() && unknown.IsNull() {
		return nil
	}
	return &schemacommon.SecurityGateBlock{
		Active:   v0.SecurityGateActive,
		Critical: critical,
		High:     high,
		Medium:   medium,
		Low:      low,
		None:     none,
		Unknown:  unknown,
	}
}

// housekeepingBlockFromV0 is the housekeeping equivalent of
// securityGateBlockFromV0.
func housekeepingBlockFromV0(v0 branchHousekeepingV0, settings client.Settings) *schemacommon.BranchHousekeepingBlock {
	keepInactiveDays := dropIfDefaultInt64(
		v0.RepositoryBranchHousekeepingKeepInactiveDays, settings.BranchHousekeepingKeepInactiveDays)
	exemptBranches := dropIfDefaultString(
		v0.RepositoryBranchHousekeepingExemptBranches, settings.BranchHousekeepingExemptBranches)

	if v0.RepositoryBranchHousekeepingActive.IsNull() &&
		keepInactiveDays.IsNull() && (exemptBranches.IsNull() || exemptBranches.ValueString() == "") {
		return nil
	}
	return &schemacommon.BranchHousekeepingBlock{
		Active:           v0.RepositoryBranchHousekeepingActive,
		KeepInactiveDays: keepInactiveDays,
		ExemptBranches:   exemptBranches,
	}
}

// dropIfDefaultInt64 returns null if value equals the instance-wide default
// -- very likely server-filled rather than practitioner-configured, and
// even if it coincidentally isn't, dropping it changes nothing about what
// gets sent to SecObserve on the next apply. See upgradeStateV0's doc
// comment.
func dropIfDefaultInt64(value types.Int64, defaultValue int64) types.Int64 {
	if value.IsNull() || value.IsUnknown() || value.ValueInt64() != defaultValue {
		return value
	}
	return types.Int64Null()
}

// dropIfDefaultString is the string equivalent of dropIfDefaultInt64.
func dropIfDefaultString(value types.String, defaultValue string) types.String {
	if value.IsNull() || value.IsUnknown() || value.ValueString() != defaultValue {
		return value
	}
	return types.StringNull()
}
