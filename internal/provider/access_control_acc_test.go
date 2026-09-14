package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccUserLifecycle(t *testing.T) {
	username := acceptanceName("user")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "secobserve_user" "test" {
  username   = %q
  first_name = "Ada"
  last_name  = "Lovelace"
  email      = "ada@example.com"
}`, username),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("secobserve_user.test", "username", username),
					// SecObserve recomputes full_name from first/last name.
					resource.TestCheckResourceAttr("secobserve_user.test", "full_name", "Ada Lovelace"),
					resource.TestCheckResourceAttr("secobserve_user.test", "is_active", "true"),
					resource.TestCheckResourceAttr("secobserve_user.test", "is_oidc_user", "false"),
				),
			},
			{
				ResourceName:      "secobserve_user.test",
				ImportState:       true,
				ImportStateId:     username,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAuthorizationGroupLifecycle(t *testing.T) {
	name := acceptanceName("group")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`resource "secobserve_authorization_group" "test" { name = %q }`, name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.TestCheckResourceAttr("secobserve_authorization_group.test", "oidc_group", ""),
			},
			{
				Config: fmt.Sprintf(`
resource "secobserve_authorization_group" "test" {
  name       = %q
  oidc_group = "platform-team"
}`, name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.TestCheckResourceAttr(
					"secobserve_authorization_group.test", "oidc_group", "platform-team"),
			},
			{
				ResourceName:      "secobserve_authorization_group.test",
				ImportState:       true,
				ImportStateId:     name,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAuthorizationGroupMemberLifecycle(t *testing.T) {
	group := acceptanceName("member-group")
	username := acceptanceName("member-user")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "secobserve_authorization_group" "test" { name = %q }
resource "secobserve_user" "test" { username = %q }

resource "secobserve_authorization_group_member" "test" {
  authorization_group = secobserve_authorization_group.test.id
  user                = secobserve_user.test.id
}`, group, username),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.TestCheckResourceAttr(
					"secobserve_authorization_group_member.test", "is_manager", "false"),
			},
			{
				// Only is_manager can change; the other two force a replace,
				// covered by TestAccAuthorizationGroupMemberIsImmutable below.
				Config: fmt.Sprintf(`
resource "secobserve_authorization_group" "test" { name = %q }
resource "secobserve_user" "test" { username = %q }

resource "secobserve_authorization_group_member" "test" {
  authorization_group = secobserve_authorization_group.test.id
  user                = secobserve_user.test.id
  is_manager          = true
}`, group, username),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(
							"secobserve_authorization_group_member.test", plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.TestCheckResourceAttr(
					"secobserve_authorization_group_member.test", "is_manager", "true"),
			},
		},
	})
}

// Setting oidc_group on the referenced group has to produce a warning; the
// testing framework can only assert on errors, not warnings, so this proves
// the write itself still succeeds despite the group being OIDC-managed --
// the actual warning text is exercised by the stub-backed unit test.
func TestAccAuthorizationGroupMemberOIDCGroupStillWrites(t *testing.T) {
	group := acceptanceName("oidc-group")
	username := acceptanceName("oidc-member")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "secobserve_authorization_group" "test" {
  name       = %q
  oidc_group = "mapped-group"
}
resource "secobserve_user" "test" { username = %q }

resource "secobserve_authorization_group_member" "test" {
  authorization_group = secobserve_authorization_group.test.id
  user                = secobserve_user.test.id
}`, group, username),
				Check: resource.TestCheckResourceAttrSet("secobserve_authorization_group_member.test", "id"),
			},
		},
	})
}

func TestAccAuthorizationGroupMemberIsImmutable(t *testing.T) {
	groupA := acceptanceName("swap-group-a")
	groupB := acceptanceName("swap-group-b")
	username := acceptanceName("swap-user")

	config := func(target string) string {
		return fmt.Sprintf(`
resource "secobserve_authorization_group" "a" { name = %q }
resource "secobserve_authorization_group" "b" { name = %q }
resource "secobserve_user" "test" { username = %q }

resource "secobserve_authorization_group_member" "test" {
  authorization_group = secobserve_authorization_group.%s.id
  user                = secobserve_user.test.id
}`, groupA, groupB, username, target)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config("a")},
			{
				Config: config("b"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(
							"secobserve_authorization_group_member.test", plancheck.ResourceActionReplace),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func TestAccAuthorizationGroupMemberRejectsInvalidIds(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "secobserve_authorization_group_member" "test" {
  authorization_group = 0
  user                = 1
}`,
				ExpectError: regexp.MustCompile(`Attribute authorization_group value must be at least 1`),
			},
		},
	})
}

func TestAccUserDataSource(t *testing.T) {
	username := acceptanceName("ds-user")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "secobserve_user" "test" { username = %q }

data "secobserve_user" "by_username" {
  username   = secobserve_user.test.username
  depends_on = [secobserve_user.test]
}`, username),
				Check: resource.TestCheckResourceAttrPair(
					"data.secobserve_user.by_username", "id", "secobserve_user.test", "id"),
			},
		},
	})
}

func TestAccAuthorizationGroupDataSource(t *testing.T) {
	name := acceptanceName("ds-group")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "secobserve_authorization_group" "test" { name = %q }

data "secobserve_authorization_group" "by_name" {
  name       = secobserve_authorization_group.test.name
  depends_on = [secobserve_authorization_group.test]
}`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.secobserve_authorization_group.by_name", "id",
						"secobserve_authorization_group.test", "id"),
					resource.TestCheckResourceAttr(
						"data.secobserve_authorization_group.by_name", "has_users", "false"),
				),
			},
		},
	})
}

func TestAccPeriodicTasksDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "secobserve_periodic_tasks" "all" {}`,
				Check:  resource.TestCheckResourceAttrSet("data.secobserve_periodic_tasks.all", "tasks.#"),
			},
		},
	})
}
