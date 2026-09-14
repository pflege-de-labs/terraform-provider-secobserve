package provider_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// Settings is a real singleton -- there is exactly one instance on the whole
// backend, shared with every other test in this package. Steps here are
// written to end on values that will not break other tests running
// concurrently against the same instance.
func TestAccSettingsLifecycle(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "secobserve_settings" "this" {
  security_gate_threshold_critical = 3
  jwt_validity_duration_user       = 200
}`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("secobserve_settings.this", "id", "1"),
					resource.TestCheckResourceAttr("secobserve_settings.this", "security_gate_threshold_critical", "3"),
					resource.TestCheckResourceAttr("secobserve_settings.this", "jwt_validity_duration_user", "200"),
					// Everything else pins to the provider's own defaults.
					resource.TestCheckResourceAttr("secobserve_settings.this", "security_gate_active", "true"),
				),
			},
			{
				ResourceName: "secobserve_settings.this",
				ImportState:  true,
				// The id given on import is ignored -- there is only one
				// settings record -- so any value works.
				ImportStateId:     "1",
				ImportStateVerify: true,
			},
			{
				// Restore defaults so this test does not leave the shared
				// singleton altered for whichever test runs next.
				Config: `resource "secobserve_settings" "this" {}`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.TestCheckResourceAttr(
					"secobserve_settings.this", "security_gate_threshold_critical", "0"),
			},
		},
	})
}

// feature_automatic_osv_scanning force-enables feature_license_management
// server-side regardless of what is sent for it in the same request
// (commons/api/views.py SettingsView.patch), unconditionally -- and
// feature_automatic_osv_scanning's own model default is true, so this fires
// even when the config never mentions it. Accepting the combination would
// apply cleanly and then fail the very next refresh with "provider produced
// inconsistent result after apply", so ValidateConfig rejects it at plan
// time instead. Both explicit-true and defaulted-true trigger it.
func TestAccSettingsOSVScanningLicenseManagementConflictRejected(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "secobserve_settings" "this" {
  feature_automatic_osv_scanning = true
  feature_license_management     = false
}`,
				ExpectError: regexp.MustCompile(
					`feature_license_management cannot be false while feature_automatic_osv_scanning is true`),
			},
			{
				// feature_automatic_osv_scanning left unset defaults to true,
				// so the same conflict fires without mentioning it at all.
				Config: `resource "secobserve_settings" "this" { feature_license_management = false }`,
				ExpectError: regexp.MustCompile(
					`feature_license_management cannot be false while feature_automatic_osv_scanning is true`),
			},
		},
	})
}

// A cleared string setting has to actually clear, and the
// observation_title_notification_status_list alias has to round-trip.
func TestAccSettingsStatusListClears(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "secobserve_settings" "this" {
  observation_title_notification_status_list = ["Open", "Affected"]
}`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.TestCheckResourceAttr(
					"secobserve_settings.this", "observation_title_notification_status_list.#", "2"),
			},
			{
				Config: `resource "secobserve_settings" "this" {}`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.TestCheckResourceAttr(
					"secobserve_settings.this", "observation_title_notification_status_list.#", "0"),
			},
		},
	})
}

// base_url_frontend is the one string setting the API will not accept an
// explicit empty string for ("This field may not be blank"), unlike every
// sibling webhook/email field, which all declare blank=True on the model.
// The provider has to send this key only when it holds a real value, and a
// later apply that leaves it out of config must leave the prior value alone
// rather than attempt to clear it -- there is no way to clear it at all.
func TestAccSettingsBaseURLFrontendCannotBeCleared(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `resource "secobserve_settings" "this" { base_url_frontend = "https://secobserve.example.com" }`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.TestCheckResourceAttr(
					"secobserve_settings.this", "base_url_frontend", "https://secobserve.example.com"),
			},
			{
				// Omitting the attribute must not attempt to clear it: the
				// prior value has to be carried forward, or every apply that
				// leaves this unset would 400.
				Config: `resource "secobserve_settings" "this" {}`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.TestCheckResourceAttr(
					"secobserve_settings.this", "base_url_frontend", "https://secobserve.example.com"),
			},
		},
	})
}
