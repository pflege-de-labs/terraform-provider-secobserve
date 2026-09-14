package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccLicensePolicyLifecycle(t *testing.T) {
	name := acceptanceName("license-policy")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "secobserve_license_policy" "test" {
  name                        = %q
  description                 = "our default policy"
  ignore_component_type_list  = ["docker", "npm"]
}`, name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("secobserve_license_policy.test", "ignore_component_type_list.#", "2"),
					// is_manager reflects an explicit License_Policy_Member row, not
					// superuser status; the acceptance provider runs as a superuser
					// and gets no such row on create (licenses/signals.py only adds
					// one for a non-superuser creator).
					resource.TestCheckResourceAttr("secobserve_license_policy.test", "is_manager", "false"),
					resource.TestCheckResourceAttr("secobserve_license_policy.test", "parent_name", ""),
				),
			},
			{
				ResourceName:      "secobserve_license_policy.test",
				ImportState:       true,
				ImportStateId:     name,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccLicensePolicyParent(t *testing.T) {
	parent := acceptanceName("license-policy-parent")
	child := acceptanceName("license-policy-child")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "secobserve_license_policy" "parent" { name = %q }

resource "secobserve_license_policy" "child" {
  name   = %q
  parent = secobserve_license_policy.parent.id
}`, parent, child),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"secobserve_license_policy.child", "parent", "secobserve_license_policy.parent", "id"),
					resource.TestCheckResourceAttr("secobserve_license_policy.child", "parent_name", parent),
				),
			},
		},
	})
}

func TestAccLicensePolicyItemLifecycle(t *testing.T) {
	policy := acceptanceName("license-policy-item")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "secobserve_license_policy" "test" { name = %q }
data "secobserve_license" "mit" { spdx_id = "MIT" }

resource "secobserve_license_policy_item" "test" {
  license_policy    = secobserve_license_policy.test.id
  license           = data.secobserve_license.mit.id
  evaluation_result = "Allowed"
}`, policy),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.TestCheckResourceAttr("secobserve_license_policy_item.test", "license_spdx_id", "MIT"),
			},
			{
				ResourceName:      "secobserve_license_policy_item.test",
				ImportStateIdFunc: licensePolicyItemImportID,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// A license_expression already in canonical SPDX form round-trips exactly:
// SecObserve's normalization is a no-op on input that's already canonical, so
// config and response agree and the plan stays empty.
func TestAccLicensePolicyItemLicenseExpressionCanonicalRoundTrips(t *testing.T) {
	policy := acceptanceName("license-policy-item-expr-ok")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "secobserve_license_policy" "test" { name = %q }

resource "secobserve_license_policy_item" "test" {
  license_policy      = secobserve_license_policy.test.id
  license_expression  = "MIT OR Apache-2.0"
  evaluation_result   = "Review required"
}`, policy),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.TestCheckResourceAttr(
					"secobserve_license_policy_item.test", "license_expression", "MIT OR Apache-2.0"),
			},
		},
	})
}

// A non-canonical license_expression is accepted and silently normalized by
// SecObserve itself, but Terraform's own plan/apply consistency check does
// not allow the applied value to differ from what was planned -- see the
// resource's schema doc for why the provider does not try to work around
// this by normalizing client-side.
func TestAccLicensePolicyItemLicenseExpressionRequiresCanonicalForm(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "secobserve_license_policy" "test" { name = "license-policy-item-expr-bad" }

resource "secobserve_license_policy_item" "test" {
  license_policy      = secobserve_license_policy.test.id
  license_expression  = "mit or apache-2.0"
  evaluation_result   = "Review required"
}`,
				ExpectError: regexp.MustCompile(`(?s)produced\s+inconsistent\s+result`),
			},
		},
	})
}

func TestAccLicensePolicyItemRequiresExactlyOneMatchField(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "secobserve_license_policy" "test" { name = "item-match-none" }

resource "secobserve_license_policy_item" "test" {
  license_policy    = secobserve_license_policy.test.id
  evaluation_result = "Allowed"
}`,
				ExpectError: regexp.MustCompile(`(?s)Missing\s+license\s+policy\s+item\s+match\s+field`),
			},
		},
	})
}

func TestAccLicensePolicyItemRejectsMultipleMatchFields(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "secobserve_license_policy" "test" { name = "item-match-multiple" }
data "secobserve_license" "mit" { spdx_id = "MIT" }

resource "secobserve_license_policy_item" "test" {
  license_policy      = secobserve_license_policy.test.id
  license             = data.secobserve_license.mit.id
  license_expression  = "MIT"
  evaluation_result   = "Allowed"
}`,
				ExpectError: regexp.MustCompile(`(?s)Multiple\s+license\s+policy\s+item\s+match\s+fields`),
			},
		},
	})
}

func TestAccLicensePolicyMemberLifecycle(t *testing.T) {
	policy := acceptanceName("license-policy-member")
	username := acceptanceName("license-policy-member-user")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "secobserve_license_policy" "test" { name = %q }
resource "secobserve_user" "test" { username = %q }

resource "secobserve_license_policy_member" "test" {
  license_policy = secobserve_license_policy.test.id
  user           = secobserve_user.test.id
}`, policy, username),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.TestCheckResourceAttr("secobserve_license_policy_member.test", "is_manager", "false"),
			},
			{
				ResourceName:      "secobserve_license_policy_member.test",
				ImportStateIdFunc: licensePolicyMemberImportID,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccLicensePolicyAuthorizationGroupMemberLifecycle(t *testing.T) {
	policy := acceptanceName("license-policy-agm")
	authGroup := acceptanceName("auth-group-lp-agm")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "secobserve_license_policy" "test" { name = %q }
resource "secobserve_authorization_group" "test" { name = %q }

resource "secobserve_license_policy_authorization_group_member" "test" {
  license_policy       = secobserve_license_policy.test.id
  authorization_group  = secobserve_authorization_group.test.id
  is_manager           = true
}`, policy, authGroup),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.TestCheckResourceAttr(
					"secobserve_license_policy_authorization_group_member.test", "is_manager", "true"),
			},
		},
	})
}

// TestAccLicensePolicyDataSource does not rely on the seeded "Standard"
// policy: on this containerized setup, initial_license_load's creation of
// "Standard" does not reliably complete during startup (import_scancode_licensedb
// -- a network call to the ScanCode LicenseDB -- appears to sometimes prevent
// create_scancode_standard_policy from ever running in the same pass; running
// it again afterward succeeds immediately). A policy this test creates itself
// is deterministic instead.
func TestAccLicensePolicyDataSource(t *testing.T) {
	name := acceptanceName("license-policy-ds")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "secobserve_license_policy" "test" { name = %q }

data "secobserve_license_policy" "by_name" {
  name       = secobserve_license_policy.test.name
  depends_on = [secobserve_license_policy.test]
}`, name),
				Check: resource.TestCheckResourceAttrPair(
					"data.secobserve_license_policy.by_name", "id", "secobserve_license_policy.test", "id"),
			},
		},
	})
}

func TestAccVEXCounterDataSource(t *testing.T) {
	t.Skip("VEX counter rows are created implicitly by exporting a CSAF/OpenVEX document; " +
		"there is no lightweight way to trigger that from an acceptance test fixture")
}

func licensePolicyItemImportID(s *terraform.State) (string, error) {
	item, ok := s.RootModule().Resources["secobserve_license_policy_item.test"]
	if !ok {
		return "", fmt.Errorf("secobserve_license_policy_item.test not found in state")
	}
	return item.Primary.ID, nil
}

func licensePolicyMemberImportID(s *terraform.State) (string, error) {
	policy, ok := s.RootModule().Resources["secobserve_license_policy.test"]
	if !ok {
		return "", fmt.Errorf("secobserve_license_policy.test not found in state")
	}
	user, ok := s.RootModule().Resources["secobserve_user.test"]
	if !ok {
		return "", fmt.Errorf("secobserve_user.test not found in state")
	}
	return fmt.Sprintf("%s/%s", policy.Primary.ID, user.Primary.ID), nil
}
