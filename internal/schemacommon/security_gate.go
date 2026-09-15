package schemacommon

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tfutil"
)

// SecurityGate is the security gate block of a product or product group.
type SecurityGate struct {
	SecurityGateActive            types.Bool  `tfsdk:"security_gate_active"`
	SecurityGateThresholdCritical types.Int64 `tfsdk:"security_gate_threshold_critical"`
	SecurityGateThresholdHigh     types.Int64 `tfsdk:"security_gate_threshold_high"`
	SecurityGateThresholdMedium   types.Int64 `tfsdk:"security_gate_threshold_medium"`
	SecurityGateThresholdLow      types.Int64 `tfsdk:"security_gate_threshold_low"`
	SecurityGateThresholdNone     types.Int64 `tfsdk:"security_gate_threshold_none"`
	SecurityGateThresholdUnknown  types.Int64 `tfsdk:"security_gate_threshold_unknown"`
}

const thresholdDescription = "Maximum number of active %s observations tolerated before the security gate fails.\n\n" +
	"Only meaningful while `security_gate_active` is `true`; SecObserve clears it otherwise. Leave it unset to " +
	"inherit the instance-wide default.\n\n" +
	"~> A value of `0` is rejected. SecObserve treats it as \"not set\" and substitutes the instance-wide " +
	"default whenever the gate is active, so it would never take effect. Use a large value such as `99999` " +
	"to ignore a severity.\n\n" +
	"~> Cannot be set while `security_gate_active` is explicitly `false`."

// thresholdSeverities maps each threshold attribute to the wording used in its
// description. Ordered so the generated documentation reads high to low.
var thresholdSeverities = []struct {
	attribute string
	severity  string
}{
	{"security_gate_threshold_critical", "critical severity"},
	{"security_gate_threshold_high", "high severity"},
	{"security_gate_threshold_medium", "medium severity"},
	{"security_gate_threshold_low", "low severity"},
	{"security_gate_threshold_none", "`None` severity"},
	{"security_gate_threshold_unknown", "unknown severity"},
}

// AddSecurityGate contributes the security gate attributes.
func AddSecurityGate(attributes map[string]schema.Attribute) {
	attributes["security_gate_active"] = schema.BoolAttribute{
		Optional: true,
		MarkdownDescription: "Whether the security gate is evaluated.\n\n" +
			"Tri-state: leave it unset to inherit from the product group or, failing that, from the instance " +
			"settings. That is different from `false`, which switches the gate off explicitly.\n\n" +
			"Setting it to `false` also clears every `security_gate_threshold_*` attribute server-side -- " +
			"each is rejected at plan time if set explicitly alongside `false`.",
	}

	for _, threshold := range thresholdSeverities {
		attributes[threshold.attribute] = schema.Int64Attribute{
			Optional:            true,
			Computed:            true,
			Validators:          []validator.Int64{int64validator.Between(0, 999999)},
			PlanModifiers:       []planmodifier.Int64{Int64UnknownWhenBoolSiblingChanges("security_gate_active")},
			MarkdownDescription: fmt.Sprintf(thresholdDescription, threshold.severity),
		}
	}
}

// ToAPI converts the block into its request representation.
func (s SecurityGate) ToAPI() client.SecurityGateFields {
	return client.SecurityGateFields{
		SecurityGateActive:            tfutil.BoolPtr(s.SecurityGateActive),
		SecurityGateThresholdCritical: tfutil.Int64Ptr(s.SecurityGateThresholdCritical),
		SecurityGateThresholdHigh:     tfutil.Int64Ptr(s.SecurityGateThresholdHigh),
		SecurityGateThresholdMedium:   tfutil.Int64Ptr(s.SecurityGateThresholdMedium),
		SecurityGateThresholdLow:      tfutil.Int64Ptr(s.SecurityGateThresholdLow),
		SecurityGateThresholdNone:     tfutil.Int64Ptr(s.SecurityGateThresholdNone),
		SecurityGateThresholdUnknown:  tfutil.Int64Ptr(s.SecurityGateThresholdUnknown),
	}
}

// FromAPI fills the block from an API response.
func (s *SecurityGate) FromAPI(fields client.SecurityGateFields) {
	s.SecurityGateActive = tfutil.Bool(fields.SecurityGateActive)
	s.SecurityGateThresholdCritical = tfutil.Int64(fields.SecurityGateThresholdCritical)
	s.SecurityGateThresholdHigh = tfutil.Int64(fields.SecurityGateThresholdHigh)
	s.SecurityGateThresholdMedium = tfutil.Int64(fields.SecurityGateThresholdMedium)
	s.SecurityGateThresholdLow = tfutil.Int64(fields.SecurityGateThresholdLow)
	s.SecurityGateThresholdNone = tfutil.Int64(fields.SecurityGateThresholdNone)
	s.SecurityGateThresholdUnknown = tfutil.Int64(fields.SecurityGateThresholdUnknown)
}

// ValidateSecurityGate rejects a threshold of 0, which SecObserve cannot
// store.
//
// The fill-in logic tests the submitted thresholds for truthiness rather than
// presence (core/api/serializers_product.py:122-134), so a 0 counts as "not
// set" and is replaced by the instance-wide default whenever the gate is
// active. Terraform forbids an applied value that differs from the planned one
// for an attribute the practitioner wrote, so accepting a 0 would fail the
// apply with "provider produced inconsistent result".
//
// Rejecting it unconditionally rather than only when security_gate_active is
// true is deliberate: the gate can also be switched on by the product group,
// which is not visible at plan time. And while the gate is off the thresholds
// are not evaluated at all, so nothing of value is lost.
func (s SecurityGate) ValidateSecurityGate(diags *diag.Diagnostics) {
	thresholds := []struct {
		attribute string
		value     types.Int64
	}{
		{"security_gate_threshold_critical", s.SecurityGateThresholdCritical},
		{"security_gate_threshold_high", s.SecurityGateThresholdHigh},
		{"security_gate_threshold_medium", s.SecurityGateThresholdMedium},
		{"security_gate_threshold_low", s.SecurityGateThresholdLow},
		{"security_gate_threshold_none", s.SecurityGateThresholdNone},
		{"security_gate_threshold_unknown", s.SecurityGateThresholdUnknown},
	}

	for _, threshold := range thresholds {
		if threshold.value.IsNull() || threshold.value.IsUnknown() || threshold.value.ValueInt64() != 0 {
			continue
		}
		diags.AddAttributeError(
			path.Root(threshold.attribute),
			"A security gate threshold of 0 cannot be stored",
			"SecObserve treats a threshold of 0 as \"not set\" and replaces it with the instance-wide "+
				"default whenever the security gate is active, so the value would never take effect.\n\n"+
				"To fail the gate on any observation of this severity, leave the attribute unset and "+
				"configure the instance-wide default instead. To ignore this severity, use a large value "+
				"such as 99999.",
		)
	}

	active := s.SecurityGateActive
	if active.IsUnknown() || active.IsNull() || active.ValueBool() {
		// Only an explicit, known false triggers SecObserve's unconditional
		// clear of every threshold; null (inherit) and true leave them alone.
		return
	}

	for _, threshold := range thresholds {
		if threshold.value.IsNull() || threshold.value.IsUnknown() {
			continue
		}
		diags.AddAttributeError(
			path.Root(threshold.attribute),
			"Cannot be set while the security gate is inactive",
			"SecObserve clears every security_gate_threshold_* attribute server-side whenever "+
				"security_gate_active is false, regardless of what is configured here.\n\n"+
				"Remove this attribute, or set security_gate_active to true or leave it unset.",
		)
	}
}
