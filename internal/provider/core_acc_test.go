package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// These run against a real SecObserve: `make up && eval "$(./test/bootstrap.sh)"`.
// The stub-backed tests in product_stub_test.go cover the drift-sensitive
// paths; these cover the round trips the stub cannot vouch for, above all that
// the payloads the provider sends are ones SecObserve actually accepts.

func TestAccProductGroupLifecycle(t *testing.T) {
	name := acceptanceName("group")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "secobserve_product_group" "test" {
  name                                  = %q
  description                           = "created by an acceptance test"
  security_gate_active                  = true
  security_gate = {
    threshold_critical = 1
  }
  observation_notification_min_severity = "High"
  observation_notification_status_list  = ["Open", "Affected"]
}`, name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("secobserve_product_group.test", "name", name),
					resource.TestCheckResourceAttr("secobserve_product_group.test", "security_gate.threshold_critical", "1"),
					resource.TestCheckResourceAttrSet("secobserve_product_group.test", "id"),
				),
			},
			{
				// Clearing a string attribute has to actually clear it.
				Config: fmt.Sprintf(`
resource "secobserve_product_group" "test" {
  name                 = %q
  security_gate_active = true
}`, name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.TestCheckResourceAttr("secobserve_product_group.test", "description", ""),
			},
			{
				ResourceName:      "secobserve_product_group.test",
				ImportState:       true,
				ImportStateId:     name,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccProductLifecycle(t *testing.T) {
	name := acceptanceName("product")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "secobserve_product" "test" {
  name              = %q
  description       = "created by an acceptance test"
  repository_prefix = "https://github.com/example/repo"
  purl              = "pkg:github/example/repo"
}`, name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("secobserve_product.test", "name", name),
					resource.TestCheckResourceAttr("secobserve_product.test", "apply_general_rules", "true"),
					// Tri-states must come back null so inheritance keeps working.
					resource.TestCheckNoResourceAttr("secobserve_product.test", "security_gate_active"),
				),
			},
			{
				Config: fmt.Sprintf(`
resource "secobserve_product" "test" {
  name                  = %q
  security_gate_active  = true
  security_gate = {
    threshold_critical = 2
  }
  propagate_branches = [{ propagate_to = "^release/.*$" }]
}`, name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("secobserve_product.test", "security_gate.threshold_critical", "2"),
					resource.TestCheckResourceAttr("secobserve_product.test", "propagate_branches.0.propagate_to", "^release/.*$"),
				),
			},
			{
				ResourceName:      "secobserve_product.test",
				ImportState:       true,
				ImportStateId:     name,
				ImportStateVerify: true,
				// security_gate/repository_branch_housekeeping are echoed
				// from config/prior state, never read back from the API
				// response (see schemacommon's plan_modifiers.go doc
				// comment) -- import has neither, so it cannot recover a
				// configured threshold. Matches the product_api_token
				// secret precedent below.
				ImportStateVerifyIgnore: []string{"security_gate.%", "security_gate.threshold_critical"},
			},
		},
	})
}

// Branch coverage focuses on the default-branch rules, which are the only
// non-obvious part: SecObserve moves the flag implicitly and refuses to delete
// the branch that holds it.
func TestAccBranchDefaultBranchHandling(t *testing.T) {
	product := acceptanceName("branch-product")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "secobserve_product" "test" {
  name = %q
}

resource "secobserve_branch" "main" {
  product           = secobserve_product.test.id
  name              = "main"
  is_default_branch = true
}

resource "secobserve_branch" "next" {
  product = secobserve_product.test.id
  name    = "next"
}`, product),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("secobserve_branch.main", "is_default_branch", "true"),
					resource.TestCheckResourceAttr("secobserve_branch.next", "is_default_branch", "false"),
				),
			},
			{
				// secobserve_product.test.repository_default_branch is set by a
				// side effect of creating secobserve_branch.main, but nothing in
				// the config makes the product depend on the branch, so a single
				// apply never re-reads the product afterwards. Reapplying the
				// same config (a no-op that still refreshes) is what surfaces it.
				Config: fmt.Sprintf(`
resource "secobserve_product" "test" {
  name = %q
}

resource "secobserve_branch" "main" {
  product           = secobserve_product.test.id
  name              = "main"
  is_default_branch = true
}

resource "secobserve_branch" "next" {
  product = secobserve_product.test.id
  name    = "next"
}`, product),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.TestCheckResourceAttrPair(
					"secobserve_product.test", "repository_default_branch",
					"secobserve_branch.main", "id"),
			},
			{
				// Moving the flag: SecObserve clears it on the old holder, so
				// a refresh has to report both branches correctly.
				Config: fmt.Sprintf(`
resource "secobserve_product" "test" {
  name = %q
}

resource "secobserve_branch" "main" {
  product = secobserve_product.test.id
  name    = "main"
}

resource "secobserve_branch" "next" {
  product           = secobserve_product.test.id
  name              = "next"
  is_default_branch = true
}`, product),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("secobserve_branch.main", "is_default_branch", "false"),
					resource.TestCheckResourceAttr("secobserve_branch.next", "is_default_branch", "true"),
				),
			},
			{
				// Terraform destroys branches independently of the product,
				// in dependency order, and SecObserve unconditionally refuses
				// to delete a default branch -- so a real `terraform destroy`
				// hits the same 400 this test would hit at teardown if a
				// branch were still marked default. Clear it first.
				Config: fmt.Sprintf(`
resource "secobserve_product" "test" {
  name = %q
}

resource "secobserve_branch" "main" {
  product = secobserve_product.test.id
  name    = "main"
}

resource "secobserve_branch" "next" {
  product = secobserve_product.test.id
  name    = "next"
}`, product),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("secobserve_branch.main", "is_default_branch", "false"),
					resource.TestCheckResourceAttr("secobserve_branch.next", "is_default_branch", "false"),
				),
			},
		},
	})
}

// Changing the product replaces the branch rather than failing with a 400.
func TestAccBranchProductIsImmutable(t *testing.T) {
	first := acceptanceName("branch-move-a")
	second := acceptanceName("branch-move-b")

	config := func(target string) string {
		return fmt.Sprintf(`
resource "secobserve_product" "a" { name = %q }
resource "secobserve_product" "b" { name = %q }

resource "secobserve_branch" "test" {
  product = secobserve_product.%s.id
  name    = "main"
}`, first, second, target)
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
						plancheck.ExpectResourceAction("secobserve_branch.test", plancheck.ResourceActionReplace),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				ResourceName:      "secobserve_branch.test",
				ImportStateIdFunc: branchImportID,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// branchImportID builds the "<product id>/<branch name>" import id from
// state, since the product's id is only known post-apply.
func branchImportID(s *terraform.State) (string, error) {
	branch, ok := s.RootModule().Resources["secobserve_branch.test"]
	if !ok {
		return "", fmt.Errorf("secobserve_branch.test not found in state")
	}
	return fmt.Sprintf("%s/%s", branch.Primary.Attributes["product"], branch.Primary.Attributes["name"]), nil
}

func TestAccServiceLifecycle(t *testing.T) {
	product := acceptanceName("service-product")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "secobserve_product" "test" { name = %q }

resource "secobserve_service" "test" {
  product = secobserve_product.test.id
  name    = "api"
}`, product),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.TestCheckResourceAttrSet("secobserve_service.test", "name_with_product"),
			},
			{
				ResourceName:      "secobserve_service.test",
				ImportStateIdFunc: serviceImportID,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// serviceImportID builds the "<product id>/<service name>" import id from
// state, since the product's id is only known post-apply.
func serviceImportID(s *terraform.State) (string, error) {
	product, ok := s.RootModule().Resources["secobserve_product.test"]
	if !ok {
		return "", fmt.Errorf("secobserve_product.test not found in state")
	}
	service, ok := s.RootModule().Resources["secobserve_service.test"]
	if !ok {
		return "", fmt.Errorf("secobserve_service.test not found in state")
	}
	return fmt.Sprintf("%s/%s", product.Primary.ID, service.Primary.Attributes["name"]), nil
}

// The provider takes a role name and sends the numeric value SecObserve
// expects; this is the round trip that proves the mapping.
func TestAccProductAPITokenRoleRoundTrip(t *testing.T) {
	product := acceptanceName("token-product")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "secobserve_product" "test" { name = %q }

resource "secobserve_product_api_token" "test" {
  product = secobserve_product.test.id
  name    = "ci"
  role    = "Upload"
}`, product),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("secobserve_product_api_token.test", "role", "Upload"),
					resource.TestCheckResourceAttrSet("secobserve_product_api_token.test", "token"),
				),
			},
			{
				// No update exists on the endpoint, so a role change replaces.
				Config: fmt.Sprintf(`
resource "secobserve_product" "test" { name = %q }

resource "secobserve_product_api_token" "test" {
  product = secobserve_product.test.id
  name    = "ci"
  role    = "Writer"
}`, product),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(
							"secobserve_product_api_token.test", plancheck.ResourceActionReplace),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				// The secret is unrecoverable: SecObserve returns it only once,
				// in the create response. Import populates everything else.
				ResourceName:            "secobserve_product_api_token.test",
				ImportStateIdFunc:       productAPITokenImportID,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token"},
			},
		},
	})
}

// productAPITokenImportID builds the "<product id>/<token name>" import id
// from state, since the product's id is only known post-apply.
func productAPITokenImportID(s *terraform.State) (string, error) {
	token, ok := s.RootModule().Resources["secobserve_product_api_token.test"]
	if !ok {
		return "", fmt.Errorf("secobserve_product_api_token.test not found in state")
	}
	return fmt.Sprintf("%s/%s", token.Primary.Attributes["product"], token.Primary.Attributes["name"]), nil
}

// An invalid role name has to fail at plan time: the API only range-checks
// 1..5 and would accept a wrong number silently.
func TestAccProductMemberRejectsUnknownRole(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "secobserve_product_member" "test" {
  product = 1
  user    = 1
  role    = "Administrator"
}`,
				ExpectError: regexp.MustCompile(`Attribute role value must be one of`),
			},
		},
	})
}

// TestAccProductMemberLifecycle uses a dedicated secobserve_user rather than
// the token's own identity: creating a product implicitly makes its creator
// an Owner member (core/signals.py), so granting a role to that same
// identity would fail as a duplicate.
func TestAccProductMemberLifecycle(t *testing.T) {
	product := acceptanceName("member-product")
	username := acceptanceName("member-user")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "secobserve_product" "test" { name = %q }
resource "secobserve_user" "test" { username = %q }

resource "secobserve_product_member" "test" {
  product = secobserve_product.test.id
  user    = secobserve_user.test.id
  role    = "Reader"
}`, product, username),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.TestCheckResourceAttr("secobserve_product_member.test", "role", "Reader"),
			},
			{
				ResourceName:      "secobserve_product_member.test",
				ImportStateIdFunc: productMemberImportID,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// productMemberImportID builds the "<product id>/<user id>" import id from
// state, since both ids are only known post-apply.
func productMemberImportID(s *terraform.State) (string, error) {
	member, ok := s.RootModule().Resources["secobserve_product_member.test"]
	if !ok {
		return "", fmt.Errorf("secobserve_product_member.test not found in state")
	}
	return fmt.Sprintf("%s/%s", member.Primary.Attributes["product"], member.Primary.Attributes["user"]), nil
}

func TestAccProductAuthorizationGroupMemberLifecycle(t *testing.T) {
	product := acceptanceName("pagm-product")
	group := acceptanceName("pagm-group")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "secobserve_product" "test" { name = %q }
resource "secobserve_authorization_group" "test" { name = %q }

resource "secobserve_product_authorization_group_member" "test" {
  product              = secobserve_product.test.id
  authorization_group  = secobserve_authorization_group.test.id
  role                 = "Reader"
}`, product, group),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.TestCheckResourceAttr(
					"secobserve_product_authorization_group_member.test", "role", "Reader"),
			},
			{
				ResourceName:      "secobserve_product_authorization_group_member.test",
				ImportStateIdFunc: productAuthorizationGroupMemberImportID,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// productAuthorizationGroupMemberImportID builds the
// "<product id>/<authorization group id>" import id from state.
func productAuthorizationGroupMemberImportID(s *terraform.State) (string, error) {
	member, ok := s.RootModule().Resources["secobserve_product_authorization_group_member.test"]
	if !ok {
		return "", fmt.Errorf("secobserve_product_authorization_group_member.test not found in state")
	}
	return fmt.Sprintf("%s/%s", member.Primary.Attributes["product"], member.Primary.Attributes["authorization_group"]), nil
}

func TestAccProductDataSource(t *testing.T) {
	name := acceptanceName("datasource-product")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "secobserve_product" "test" { name = %q }

data "secobserve_product" "by_name" {
  name       = secobserve_product.test.name
  depends_on = [secobserve_product.test]
}`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.secobserve_product.by_name", "id", "secobserve_product.test", "id"),
					resource.TestCheckResourceAttrSet(
						"data.secobserve_product.by_name", "has_component"),
				),
			},
		},
	})
}

// Supplying neither or both lookup keys is a configuration error.
func TestAccProductDataSourceRequiresExactlyOneKey(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// datasourcevalidator.ExactlyOneOf's diagnostic summary is
				// "Missing Attribute Configuration" when nothing is set.
				Config:      `data "secobserve_product" "test" {}`,
				ExpectError: regexp.MustCompile(`Missing Attribute Configuration`),
			},
			{
				// ExactlyOneOf's summary when both are set is "Invalid
				// Attribute Combination".
				Config: `
data "secobserve_product" "test" {
  id   = 1
  name = "x"
}`,
				ExpectError: regexp.MustCompile(`Invalid Attribute Combination`),
			},
		},
	})
}
