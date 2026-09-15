package schemacommon

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// The two functions below reproduce the pre-v1 flat schema shape, for use
// only as a resource.StateUpgrader's PriorSchema when upgrading state written
// by a provider version before the security_gate/repository_branch_housekeeping
// nested attributes existed. Validators, Defaults and PlanModifiers are
// irrelevant to interpreting already-written state and are deliberately
// omitted -- only the attribute names, types and Optional/Computed markers
// need to match what was actually written. Do not use these for anything but
// a StateUpgrader.

// AddSecurityGateV0 reproduces the pre-v1 flat security_gate_threshold_*
// attributes.
func AddSecurityGateV0(attributes map[string]schema.Attribute) {
	attributes["security_gate_active"] = schema.BoolAttribute{Optional: true}
	for _, name := range []string{
		"security_gate_threshold_critical",
		"security_gate_threshold_high",
		"security_gate_threshold_medium",
		"security_gate_threshold_low",
		"security_gate_threshold_none",
		"security_gate_threshold_unknown",
	} {
		attributes[name] = schema.Int64Attribute{Optional: true, Computed: true}
	}
}

// AddBranchHousekeepingV0 reproduces the pre-v1 flat
// repository_branch_housekeeping_* attributes.
func AddBranchHousekeepingV0(attributes map[string]schema.Attribute) {
	attributes["repository_branch_housekeeping_active"] = schema.BoolAttribute{Optional: true}
	attributes["repository_branch_housekeeping_keep_inactive_days"] = schema.Int64Attribute{Optional: true, Computed: true}
	attributes["repository_branch_housekeeping_exempt_branches"] = schema.StringAttribute{Optional: true, Computed: true}
}

// The two functions below reproduce the shipped v1 shape (top-level
// *_active bool + a SingleNestedAttribute holding only the thresholds/
// settings, no `active` inside), for use as a resource.StateUpgrader's
// PriorSchema when upgrading state written by that version -- before
// security_gate/repository_branch_housekeeping became real Blocks with
// `active` folded in. Same decode-only caveat as the V0 functions above.

// AddSecurityGateV1 reproduces the shipped v1 security_gate_active +
// security_gate (nested attribute, thresholds only) shape.
func AddSecurityGateV1(attributes map[string]schema.Attribute) {
	attributes["security_gate_active"] = schema.BoolAttribute{Optional: true}
	thresholdAttributes := map[string]schema.Attribute{}
	for _, name := range []string{
		"threshold_critical", "threshold_high", "threshold_medium",
		"threshold_low", "threshold_none", "threshold_unknown",
	} {
		thresholdAttributes[name] = schema.Int64Attribute{Optional: true}
	}
	attributes["security_gate"] = schema.SingleNestedAttribute{Optional: true, Attributes: thresholdAttributes}
}

// AddBranchHousekeepingV1 reproduces the shipped v1
// repository_branch_housekeeping_active + repository_branch_housekeeping
// (nested attribute, keep_inactive_days/exempt_branches only) shape.
func AddBranchHousekeepingV1(attributes map[string]schema.Attribute) {
	attributes["repository_branch_housekeeping_active"] = schema.BoolAttribute{Optional: true}
	attributes["repository_branch_housekeeping"] = schema.SingleNestedAttribute{
		Optional: true,
		Attributes: map[string]schema.Attribute{
			"keep_inactive_days": schema.Int64Attribute{Optional: true},
			"exempt_branches":    schema.StringAttribute{Optional: true},
		},
	}
}
