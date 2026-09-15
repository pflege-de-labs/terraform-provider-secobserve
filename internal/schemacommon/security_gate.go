package schemacommon

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tfutil"
)

// SecurityGate is the security gate block of a product or product group.
type SecurityGate struct {
	SecurityGateActive types.Bool              `tfsdk:"security_gate_active"`
	Thresholds         *SecurityGateThresholds `tfsdk:"security_gate"`
}

// SecurityGateThresholds is the nested security_gate object. It deliberately
// carries no server-filled values: state is built from the plan, never from
// the API response -- see the package doc comment on plan_modifiers.go for
// why, and ValidateSecurityGate below for what SecObserve does when
// security_gate_active is false.
type SecurityGateThresholds struct {
	Critical types.Int64 `tfsdk:"threshold_critical"`
	High     types.Int64 `tfsdk:"threshold_high"`
	Medium   types.Int64 `tfsdk:"threshold_medium"`
	Low      types.Int64 `tfsdk:"threshold_low"`
	None     types.Int64 `tfsdk:"threshold_none"`
	Unknown  types.Int64 `tfsdk:"threshold_unknown"`
}

const thresholdDescription = "Maximum number of active %s observations tolerated before the security gate fails.\n\n" +
	"~> A value of `0` is rejected. SecObserve treats it as \"not set\" and substitutes the instance-wide " +
	"default whenever the gate is active, so it would never take effect. Use a large value such as `99999` " +
	"to ignore a severity."

// thresholdSeverities maps each threshold attribute to the wording used in its
// description. Ordered so the generated documentation reads high to low.
var thresholdSeverities = []struct {
	attribute string
	severity  string
}{
	{"threshold_critical", "critical severity"},
	{"threshold_high", "high severity"},
	{"threshold_medium", "medium severity"},
	{"threshold_low", "low severity"},
	{"threshold_none", "`None` severity"},
	{"threshold_unknown", "unknown severity"},
}

// AddSecurityGate contributes the security gate attributes.
func AddSecurityGate(attributes map[string]schema.Attribute) {
	attributes["security_gate_active"] = schema.BoolAttribute{
		Optional: true,
		MarkdownDescription: "Whether the security gate is evaluated.\n\n" +
			"Tri-state: leave it unset to inherit from the product group or, failing that, from the instance " +
			"settings (which defaults to `true`). That is different from `false`, which switches the gate off " +
			"explicitly.\n\n" +
			"~> If this product belongs to a product group, an explicit `true`/`false` on the **product group** " +
			"always wins over this attribute -- this product's own value only applies when the product group's " +
			"is left unset. See `docs/design/api-quirks.md` for the source reference.\n\n" +
			"Setting it to `false` also clears every threshold in `security_gate` server-side -- the block is " +
			"rejected at plan time if set alongside `false`.",
	}

	thresholdAttributes := map[string]schema.Attribute{}
	for _, threshold := range thresholdSeverities {
		thresholdAttributes[threshold.attribute] = schema.Int64Attribute{
			Optional:            true,
			Validators:          []validator.Int64{int64validator.Between(0, 999999)},
			MarkdownDescription: fmt.Sprintf(thresholdDescription, threshold.severity),
		}
	}
	attributes["security_gate"] = schema.SingleNestedAttribute{
		Optional:   true,
		Attributes: thresholdAttributes,
		MarkdownDescription: "Thresholds for the security gate. Only meaningful while `security_gate_active` " +
			"is `true`; SecObserve clears every threshold server-side otherwise, and this attribute is rejected " +
			"at plan time if set alongside `security_gate_active = false`.\n\n" +
			"Leave a threshold unset to inherit the instance-wide default for that severity -- the default is " +
			"not reflected back into this block, matching the rest of this provider's read-only/informational " +
			"data being reserved for data sources.\n\n" +
			"~> Unlike every other attribute in this provider, this block's state is echoed from your " +
			"configuration rather than read back from SecObserve. Two consequences: `terraform import` cannot " +
			"recover configured thresholds (add them to your configuration afterwards to match what's actually " +
			"configured), and changing a threshold directly in SecObserve rather than through Terraform will " +
			"not be detected as drift.",
	}
}

// ToAPI converts the block into its request representation.
func (s SecurityGate) ToAPI() client.SecurityGateFields {
	fields := client.SecurityGateFields{
		SecurityGateActive: tfutil.BoolPtr(s.SecurityGateActive),
	}
	if s.Thresholds != nil {
		fields.SecurityGateThresholdCritical = tfutil.Int64Ptr(s.Thresholds.Critical)
		fields.SecurityGateThresholdHigh = tfutil.Int64Ptr(s.Thresholds.High)
		fields.SecurityGateThresholdMedium = tfutil.Int64Ptr(s.Thresholds.Medium)
		fields.SecurityGateThresholdLow = tfutil.Int64Ptr(s.Thresholds.Low)
		fields.SecurityGateThresholdNone = tfutil.Int64Ptr(s.Thresholds.None)
		fields.SecurityGateThresholdUnknown = tfutil.Int64Ptr(s.Thresholds.Unknown)
	}
	return fields
}

// FromAPI fills the block from an API response.
//
// Deliberately does NOT populate Thresholds: unlike the rest of this
// provider, security_gate's state is carried forward from the plan/prior
// state by the resource's Create/Read/Update, not read back from the
// response. See the package doc comment on plan_modifiers.go.
func (s *SecurityGate) FromAPI(fields client.SecurityGateFields) {
	s.SecurityGateActive = tfutil.Bool(fields.SecurityGateActive)
}

// ValidateSecurityGate rejects a threshold of 0, which SecObserve cannot
// store, and rejects/warns about the security_gate block's relationship with
// security_gate_active.
//
// The fill-in logic tests the submitted thresholds for truthiness rather than
// presence (core/api/serializers_product.py:122-134), so a 0 counts as "not
// set" and is replaced by the instance-wide default whenever the gate is
// active. Terraform forbids an applied value that differs from the planned one
// for an attribute the practitioner wrote, so accepting a 0 would fail the
// apply with "provider produced inconsistent result".
func (s SecurityGate) ValidateSecurityGate(diags *diag.Diagnostics) {
	if s.Thresholds == nil {
		return
	}

	thresholds := []struct {
		attribute string
		value     types.Int64
	}{
		{"threshold_critical", s.Thresholds.Critical},
		{"threshold_high", s.Thresholds.High},
		{"threshold_medium", s.Thresholds.Medium},
		{"threshold_low", s.Thresholds.Low},
		{"threshold_none", s.Thresholds.None},
		{"threshold_unknown", s.Thresholds.Unknown},
	}

	for _, threshold := range thresholds {
		if threshold.value.IsNull() || threshold.value.IsUnknown() || threshold.value.ValueInt64() != 0 {
			continue
		}
		diags.AddAttributeError(
			path.Root("security_gate").AtName(threshold.attribute),
			"A security gate threshold of 0 cannot be stored",
			"SecObserve treats a threshold of 0 as \"not set\" and replaces it with the instance-wide "+
				"default whenever the security gate is active, so the value would never take effect.\n\n"+
				"To fail the gate on any observation of this severity, leave the attribute unset and "+
				"configure the instance-wide default instead. To ignore this severity, use a large value "+
				"such as 99999.",
		)
	}

	active := s.SecurityGateActive
	if active.IsUnknown() {
		return
	}

	if !active.IsNull() && !active.ValueBool() {
		diags.AddAttributeError(
			path.Root("security_gate"),
			"Cannot be set while the security gate is inactive",
			"SecObserve clears every threshold server-side whenever security_gate_active is false, "+
				"regardless of what is configured here.\n\n"+
				"Remove this block, or set security_gate_active to true.",
		)
		return
	}

	if active.IsNull() {
		diags.AddAttributeWarning(
			path.Root("security_gate"),
			"Ignored unless security_gate_active is true",
			"security_gate_active is unset here, so SecObserve never consults these thresholds on this "+
				"resource -- whatever ends up active comes from the product group or the instance-wide default "+
				"instead, using their own thresholds. Set security_gate_active = true to make this resource's "+
				"thresholds apply.",
		)
	}
}
