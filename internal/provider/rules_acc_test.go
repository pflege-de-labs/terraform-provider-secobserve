package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccGeneralRuleLifecycle(t *testing.T) {
	name := acceptanceName("general-rule")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "secobserve_general_rule" "test" {
  name        = %q
  description = "ignore dev dependencies"
  title       = "^dev-.*"
  new_status  = "Resolved"
}`, name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("secobserve_general_rule.test", "type", "Fields"),
					resource.TestCheckResourceAttr("secobserve_general_rule.test", "enabled", "true"),
					// feature_general_rules_need_approval defaults to false,
					// so a freshly created rule is auto-approved.
					resource.TestCheckResourceAttr("secobserve_general_rule.test", "approval_status", "Auto approved"),
					resource.TestCheckResourceAttrSet("secobserve_general_rule.test", "user"),
				),
			},
			{
				ResourceName:      "secobserve_general_rule.test",
				ImportState:       true,
				ImportStateId:     name,
				ImportStateVerify: true,
			},
		},
	})
}

// The model declares description as blank=True, but the serializer's
// validate_description rejects an empty value outright ("Must be set").
// Required + a length validator catches this at plan time.
func TestAccGeneralRuleRequiresDescription(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "secobserve_general_rule" "test" {
  name        = "x"
  description = ""
}`,
				ExpectError: regexp.MustCompile(`(?s)Attribute description.*string length must be between 1 and 2048`),
			},
		},
	})
}

func TestAccGeneralRuleRegoRequiresModule(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "secobserve_general_rule" "test" {
  name        = "rego-rule"
  description = "needs a module"
  type        = "Rego"
}`,
				ExpectError: regexp.MustCompile(`rego_module is required when type is "Rego"`),
			},
		},
	})
}

func TestAccGeneralRuleEmptyVEXRemediationsRejected(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "secobserve_general_rule" "test" {
  name                  = "empty-remediations"
  description            = "test"
  new_vex_remediations   = []
}`,
				ExpectError: regexp.MustCompile(`Empty new_vex_remediations list`),
			},
		},
	})
}

func TestAccGeneralRuleVEXRemediationsRoundTrip(t *testing.T) {
	name := acceptanceName("vex-remediation-rule")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "secobserve_general_rule" "test" {
  name        = %q
  description = "attach a remediation"

  new_vex_remediations = [
    { category = "workaround", text = "upgrade to 2.0" },
  ]
}`, name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("secobserve_general_rule.test", "new_vex_remediations.#", "1"),
					resource.TestCheckResourceAttr(
						"secobserve_general_rule.test", "new_vex_remediations.0.category", "workaround"),
				),
			},
		},
	})
}

func TestAccGeneralRuleRequiresSuperuser(t *testing.T) {
	// Covered structurally: UserHasGeneralRulePermission rejects any non-GET
	// request from a non-superuser (rules/api/permissions.py:15-26). The
	// provider itself requires a superuser token at Configure time, so this
	// is exercised implicitly by every other test in this file -- a
	// non-superuser token would fail Configure before ever reaching here.
	t.Skip("provider Configure already requires a superuser token; see TestVerifyInstance")
}

func TestAccProductRuleLifecycle(t *testing.T) {
	product := acceptanceName("rule-product")
	name := acceptanceName("product-rule")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "secobserve_product" "test" { name = %q }

resource "secobserve_product_rule" "test" {
  product     = secobserve_product.test.id
  name        = %q
  description = "resolve false positives from this scanner"
  new_status  = "False positive"
}`, product, name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("secobserve_product_rule.test", "product"),
					resource.TestCheckResourceAttr("secobserve_product_rule.test", "approval_status", "Auto approved"),
				),
			},
			{
				ResourceName:      "secobserve_product_rule.test",
				ImportStateIdFunc: productRuleImportID,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccProductRuleProductIsImmutable(t *testing.T) {
	productA := acceptanceName("rule-move-a")
	productB := acceptanceName("rule-move-b")

	config := func(target string) string {
		return fmt.Sprintf(`
resource "secobserve_product" "a" { name = %q }
resource "secobserve_product" "b" { name = %q }

resource "secobserve_product_rule" "test" {
  product     = secobserve_product.%s.id
  name        = "move-me"
  description = "test"
}`, productA, productB, target)
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
						plancheck.ExpectResourceAction("secobserve_product_rule.test", plancheck.ResourceActionReplace),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

// productRuleImportID builds the "<product id>/<rule name>" import id from
// state, since the product's id is only known post-apply.
func productRuleImportID(s *terraform.State) (string, error) {
	product, ok := s.RootModule().Resources["secobserve_product.test"]
	if !ok {
		return "", fmt.Errorf("secobserve_product.test not found in state")
	}
	rule, ok := s.RootModule().Resources["secobserve_product_rule.test"]
	if !ok {
		return "", fmt.Errorf("secobserve_product_rule.test not found in state")
	}
	return fmt.Sprintf("%s/%s", product.Primary.ID, rule.Primary.Attributes["name"]), nil
}
