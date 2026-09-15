package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccLicenseDataSource relies on SecObserve's own startup seeding
// (docker/backend/prod/django/entrypoint runs "manage.py initial_license_load"
// on first boot), so MIT is expected to exist on any fresh instance.
func TestAccLicenseDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "secobserve_license" "mit" {
  spdx_id = "MIT"
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.secobserve_license.mit", "spdx_id", "MIT"),
					resource.TestCheckResourceAttrSet("data.secobserve_license.mit", "id"),
					resource.TestCheckResourceAttrSet("data.secobserve_license.mit", "name"),
				),
			},
		},
	})
}

func TestAccLicenseGroupLifecycle(t *testing.T) {
	name := acceptanceName("license-group")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
data "secobserve_license" "mit" { spdx_id = "MIT" }

resource "secobserve_license_group" "test" {
  name        = %q
  description = "permissive licenses we allow"
  licenses    = [data.secobserve_license.mit.id]
}`, name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("secobserve_license_group.test", "licenses.#", "1"),
					resource.TestCheckResourceAttr("secobserve_license_group.test", "has_licenses", "true"),
					// is_manager reflects an explicit License_Group_Member row, not
					// superuser status; see the matching note on
					// TestAccLicensePolicyLifecycle.
					resource.TestCheckResourceAttr("secobserve_license_group.test", "is_manager", "false"),
				),
			},
			{
				// Reconciles licenses via add_license/remove_license: MIT
				// removed, Apache-2.0 added.
				Config: fmt.Sprintf(`
data "secobserve_license" "apache" { spdx_id = "Apache-2.0" }

resource "secobserve_license_group" "test" {
  name        = %q
  description = "permissive licenses we allow"
  licenses    = [data.secobserve_license.apache.id]
}`, name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("secobserve_license_group.test", "licenses.#", "1"),
				),
			},
			{
				ResourceName:      "secobserve_license_group.test",
				ImportState:       true,
				ImportStateId:     name,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccLicenseGroupMemberLifecycle(t *testing.T) {
	group := acceptanceName("license-group-member")
	username := acceptanceName("license-group-member-user")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "secobserve_license_group" "test" { name = %q }
resource "secobserve_user" "test" { username = %q }

resource "secobserve_license_group_member" "test" {
  license_group = secobserve_license_group.test.id
  user          = secobserve_user.test.id
}`, group, username),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.TestCheckResourceAttr("secobserve_license_group_member.test", "is_manager", "false"),
			},
			{
				ResourceName:      "secobserve_license_group_member.test",
				ImportStateIdFunc: licenseGroupMemberImportID,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccLicenseGroupAuthorizationGroupMemberLifecycle(t *testing.T) {
	licenseGroup := acceptanceName("license-group-agm")
	authGroup := acceptanceName("auth-group-agm")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "secobserve_license_group" "test" { name = %q }
resource "secobserve_authorization_group" "test" { name = %q }

resource "secobserve_license_group_authorization_group_member" "test" {
  license_group       = secobserve_license_group.test.id
  authorization_group = secobserve_authorization_group.test.id
  is_manager          = true
}`, licenseGroup, authGroup),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.TestCheckResourceAttr(
					"secobserve_license_group_authorization_group_member.test", "is_manager", "true"),
			},
			{
				ResourceName:      "secobserve_license_group_authorization_group_member.test",
				ImportStateIdFunc: licenseGroupAuthorizationGroupMemberImportID,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// licenseGroupAuthorizationGroupMemberImportID builds the
// "<license group id>/<authorization group id>" import id from state.
func licenseGroupAuthorizationGroupMemberImportID(s *terraform.State) (string, error) {
	member, ok := s.RootModule().Resources["secobserve_license_group_authorization_group_member.test"]
	if !ok {
		return "", fmt.Errorf("secobserve_license_group_authorization_group_member.test not found in state")
	}
	return fmt.Sprintf("%s/%s", member.Primary.Attributes["license_group"], member.Primary.Attributes["authorization_group"]), nil
}

func licenseGroupMemberImportID(s *terraform.State) (string, error) {
	group, ok := s.RootModule().Resources["secobserve_license_group.test"]
	if !ok {
		return "", fmt.Errorf("secobserve_license_group.test not found in state")
	}
	user, ok := s.RootModule().Resources["secobserve_user.test"]
	if !ok {
		return "", fmt.Errorf("secobserve_user.test not found in state")
	}
	return fmt.Sprintf("%s/%s", group.Primary.ID, user.Primary.ID), nil
}
