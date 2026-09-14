package schemacommon

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tfutil"
)

// RiskAcceptance is the risk acceptance expiry block of a product or product
// group.
type RiskAcceptance struct {
	RiskAcceptanceExpiryActive types.Bool  `tfsdk:"risk_acceptance_expiry_active"`
	RiskAcceptanceExpiryDays   types.Int64 `tfsdk:"risk_acceptance_expiry_days"`
}

// AddRiskAcceptance contributes the risk acceptance expiry attributes.
func AddRiskAcceptance(attributes map[string]schema.Attribute) {
	attributes["risk_acceptance_expiry_active"] = schema.BoolAttribute{
		Optional: true,
		MarkdownDescription: "Whether accepted risks expire automatically.\n\n" +
			"Tri-state: leave it unset to inherit from the product group or, failing that, from the instance " +
			"settings. That is different from `false`, which disables expiry explicitly.",
	}
	attributes["risk_acceptance_expiry_days"] = schema.Int64Attribute{
		Optional:   true,
		Validators: []validator.Int64{int64validator.Between(0, 999999)},
		MarkdownDescription: "Days before an accepted risk expires. `0` means accepted risks never expire. " +
			"Leave it unset to inherit the instance-wide default.",
	}
}

// ToAPI converts the block into its request representation.
func (r RiskAcceptance) ToAPI() client.RiskAcceptanceFields {
	return client.RiskAcceptanceFields{
		RiskAcceptanceExpiryActive: tfutil.BoolPtr(r.RiskAcceptanceExpiryActive),
		RiskAcceptanceExpiryDays:   tfutil.Int64Ptr(r.RiskAcceptanceExpiryDays),
	}
}

// FromAPI fills the block from an API response.
func (r *RiskAcceptance) FromAPI(fields client.RiskAcceptanceFields) {
	r.RiskAcceptanceExpiryActive = tfutil.Bool(fields.RiskAcceptanceExpiryActive)
	r.RiskAcceptanceExpiryDays = tfutil.Int64(fields.RiskAcceptanceExpiryDays)
}

// LicensePolicy links a product or group to a license policy.
type LicensePolicy struct {
	LicensePolicy types.Int64 `tfsdk:"license_policy"`
}

// AddLicensePolicy contributes the license policy attribute.
func AddLicensePolicy(attributes map[string]schema.Attribute) {
	attributes["license_policy"] = schema.Int64Attribute{
		Optional:   true,
		Validators: []validator.Int64{int64validator.AtLeast(1)},
		MarkdownDescription: "Id of the license policy to apply. Use the `secobserve_license_policy` data " +
			"source to resolve a policy name.\n\n" +
			"~> SecObserve protects a policy that is still referenced, so deleting one while it is in use " +
			"fails with a conflict.",
	}
}

// ToAPI converts the block into its request representation.
func (l LicensePolicy) ToAPI() client.LicensePolicyFields {
	return client.LicensePolicyFields{LicensePolicy: tfutil.Int64Ptr(l.LicensePolicy)}
}

// FromAPI fills the block from an API response.
func (l *LicensePolicy) FromAPI(fields client.LicensePolicyFields) {
	l.LicensePolicy = tfutil.Int64(fields.LicensePolicy)
}
