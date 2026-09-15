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
