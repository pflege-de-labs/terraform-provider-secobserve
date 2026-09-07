package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
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
  security_gate_threshold_critical      = 1
  observation_notification_min_severity = "High"
  observation_notification_status_list  = ["Open", "Affected"]
}`, name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("secobserve_product_group.test", "name", name),
					resource.TestCheckResourceAttr("secobserve_product_group.test", "security_gate_threshold_critical", "1"),
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
  name                             = %q
  security_gate_active             = true
  security_gate_threshold_critical = 2
  propagate_branches               = [{ propagate_to = "^release/.*$" }]
}`, name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("secobserve_product.test", "security_gate_threshold_critical", "2"),
					resource.TestCheckResourceAttr("secobserve_product.test", "propagate_branches.0.propagate_to", "^release/.*$"),
				),
			},
			{
				ResourceName:      "secobserve_product.test",
				ImportState:       true,
				ImportStateId:     name,
				ImportStateVerify: true,
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
					// The product picks the default branch up as a computed value.
					resource.TestCheckResourceAttrPair(
						"secobserve_product.test", "repository_default_branch",
						"secobserve_branch.main", "id"),
				),
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
		},
	})
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
		},
	})
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
		},
	})
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
				Config:      `data "secobserve_product" "test" {}`,
				ExpectError: regexp.MustCompile(`Invalid Attribute Combination`),
			},
			{
				Config:      `data "secobserve_product" "test" { id = 1, name = "x" }`,
				ExpectError: regexp.MustCompile(`Invalid Attribute Combination|Argument or block definition required`),
			},
		},
	})
}
