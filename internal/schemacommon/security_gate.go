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

// SecurityGate is the security_gate block of a product or product group.
type SecurityGate struct {
	Block *SecurityGateBlock `tfsdk:"security_gate"`
}

// SecurityGateBlock is the nested security_gate block. Absent entirely means
// inherit from the product group or the instance settings (tri-state); a
// present block with no `active` means active -- setting any threshold
// implies activating the gate. `active = false` inside the block is the
// explicit way to switch it off.
//
// The block deliberately carries no server-filled values: state is built
// from the plan, never from the API response -- see the package doc comment
// on plan_modifiers.go for why, and ValidateSecurityGate below for what
// SecObserve does when the gate is inactive.
type SecurityGateBlock struct {
	Active   types.Bool  `tfsdk:"active"`
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

// AddSecurityGate contributes the security_gate block.
func AddSecurityGate(blocks map[string]schema.Block) {
	thresholdAttributes := map[string]schema.Attribute{
		"active": schema.BoolAttribute{
			Optional: true,
			MarkdownDescription: "Explicitly switches the gate off when set to `false`. Leave unset (or " +
				"`true`) to activate the gate -- the block's mere presence already does that, this exists so " +
				"the block can also express \"explicitly off\" without being removed.\n\n" +
				"~> If this product belongs to a product group, an explicit `true`/`false` on the **product " +
				"group** always wins over this one -- this product's own value only applies when the product " +
				"group's is left unset. See `docs/design/api-quirks.md` for the source reference.",
		},
	}
	for _, threshold := range thresholdSeverities {
		thresholdAttributes[threshold.attribute] = schema.Int64Attribute{
			Optional:            true,
			Validators:          []validator.Int64{int64validator.Between(0, 999999)},
			MarkdownDescription: fmt.Sprintf(thresholdDescription, threshold.severity),
		}
	}
	blocks["security_gate"] = schema.SingleNestedBlock{
		Attributes: thresholdAttributes,
		MarkdownDescription: "Security gate configuration. Omit this block entirely to inherit from the " +
			"product group or, failing that, from the instance settings (which defaults to active). Writing " +
			"the block -- even empty -- activates the gate; set `active = false` inside it to switch the gate " +
			"off explicitly instead of inheriting.\n\n" +
			"SecObserve clears every threshold server-side whenever the gate ends up inactive, so a threshold " +
			"is rejected at plan time if set alongside `active = false`.\n\n" +
			"Leave a threshold unset to inherit the instance-wide default for that severity -- the default is " +
			"not reflected back into this block, matching the rest of this provider's read-only/informational " +
			"data being reserved for data sources.\n\n" +
			"~> Unlike every other attribute in this provider, this block's state is echoed from your " +
			"configuration rather than read back from SecObserve. Two consequences: `terraform import` cannot " +
			"recover a configured gate (add the block to your configuration afterwards to match what's " +
			"actually configured), and changing it directly in SecObserve rather than through Terraform will " +
			"not be detected as drift.",
	}
}

// ToAPI converts the block into its request representation. A present block
// with `active` unset resolves to active = true; thresholds are sent as
// configured regardless, since ValidateSecurityGate is what rejects a
// threshold alongside an explicit active = false.
func (s SecurityGate) ToAPI() client.SecurityGateFields {
	if s.Block == nil {
		return client.SecurityGateFields{}
	}

	active := true
	if !s.Block.Active.IsNull() && !s.Block.Active.IsUnknown() {
		active = s.Block.Active.ValueBool()
	}

	return client.SecurityGateFields{
		SecurityGateActive:            &active,
		SecurityGateThresholdCritical: tfutil.Int64Ptr(s.Block.Critical),
		SecurityGateThresholdHigh:     tfutil.Int64Ptr(s.Block.High),
		SecurityGateThresholdMedium:   tfutil.Int64Ptr(s.Block.Medium),
		SecurityGateThresholdLow:      tfutil.Int64Ptr(s.Block.Low),
		SecurityGateThresholdNone:     tfutil.Int64Ptr(s.Block.None),
		SecurityGateThresholdUnknown:  tfutil.Int64Ptr(s.Block.Unknown),
	}
}

// FromAPI intentionally does nothing: security_gate's state is carried
// forward from the plan/prior state by the resource's Create/Read/Update, not
// read back from the response. See the package doc comment on
// plan_modifiers.go.
func (s *SecurityGate) FromAPI(client.SecurityGateFields) {}

// ValidateSecurityGate rejects a threshold of 0, which SecObserve cannot
// store, and rejects a threshold set alongside an explicit active = false.
//
// The fill-in logic tests the submitted thresholds for truthiness rather than
// presence (core/api/serializers_product.py:122-134), so a 0 counts as "not
// set" and is replaced by the instance-wide default whenever the gate is
// active. Terraform forbids an applied value that differs from the planned one
// for an attribute the practitioner wrote, so accepting a 0 would fail the
// apply with "provider produced inconsistent result".
func (s SecurityGate) ValidateSecurityGate(diags *diag.Diagnostics) {
	if s.Block == nil {
		return
	}

	thresholds := []struct {
		attribute string
		value     types.Int64
	}{
		{"threshold_critical", s.Block.Critical},
		{"threshold_high", s.Block.High},
		{"threshold_medium", s.Block.Medium},
		{"threshold_low", s.Block.Low},
		{"threshold_none", s.Block.None},
		{"threshold_unknown", s.Block.Unknown},
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

	active := s.Block.Active
	if active.IsUnknown() || active.IsNull() || active.ValueBool() {
		// Block present with active unset or true: the gate is on, every
		// threshold is meaningful.
		return
	}

	for _, threshold := range thresholds {
		if threshold.value.IsNull() || threshold.value.IsUnknown() {
			continue
		}
		diags.AddAttributeError(
			path.Root("security_gate").AtName(threshold.attribute),
			"Cannot be set while the security gate is inactive",
			"SecObserve clears every threshold server-side whenever the security gate is inactive, "+
				"regardless of what is configured here.\n\n"+
				"Remove this attribute, or remove active = false from the security_gate block.",
		)
	}
}
