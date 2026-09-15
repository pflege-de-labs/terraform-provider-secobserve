package settings

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// ValidateConfig rejects a configuration that asks for
// feature_license_management = false while feature_automatic_osv_scanning is
// (or defaults to) true.
//
// SecObserve force-enables feature_license_management server-side whenever
// feature_automatic_osv_scanning is true, unconditionally, regardless of what
// was submitted for it in the same request (commons/api/views.py
// SettingsView.patch). Accepting the combination here would apply cleanly but
// fail on the very next refresh with "provider produced inconsistent result
// after apply", because feature_automatic_osv_scanning's own model default is
// also true (commons/models.py:216) -- so this bites even a config that never
// mentions it.
func (r *settingsResource) ValidateConfig(
	ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse,
) {
	var config model
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.FeatureAutomaticOSVScanning.IsUnknown() || config.FeatureLicenseManagement.IsUnknown() {
		// Unknown (e.g. from a variable, unresolved during `terraform
		// validate`) must not be treated as null/false -- defer to
		// SecObserve's own behavior as the backstop once both are known.
		return
	}

	osvScanningEffectivelyTrue := config.FeatureAutomaticOSVScanning.IsNull() ||
		config.FeatureAutomaticOSVScanning.ValueBool()
	licenseManagementExplicitlyFalse := !config.FeatureLicenseManagement.IsNull() &&
		!config.FeatureLicenseManagement.ValueBool()

	if osvScanningEffectivelyTrue && licenseManagementExplicitlyFalse {
		resp.Diagnostics.AddAttributeError(
			path.Root("feature_license_management"),
			"feature_license_management cannot be false while feature_automatic_osv_scanning is true",
			"SecObserve force-enables feature_license_management whenever feature_automatic_osv_scanning "+
				"is true, regardless of what is configured here. feature_automatic_osv_scanning defaults "+
				"to true, so this also applies when it is left unset entirely.\n\n"+
				"Set feature_automatic_osv_scanning = false, or remove feature_license_management = false "+
				"from the configuration.",
		)
	}
}
