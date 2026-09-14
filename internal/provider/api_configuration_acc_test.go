package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccAPIConfigurationLifecycle(t *testing.T) {
	product := acceptanceName("api-config-product")
	name := acceptanceName("api-config")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "secobserve_product" "test" { name = %q }

data "secobserve_parser" "trivy" {
  name = "Trivy Operator Prometheus"
}

resource "secobserve_api_configuration" "test" {
  product  = secobserve_product.test.id
  name     = %q
  parser   = data.secobserve_parser.trivy.id
  base_url = "https://trivy.example.com"
  api_key  = "secret-key"
}`, product, name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("secobserve_api_configuration.test", "name", name),
					// api_key round-trips for a superuser token.
					resource.TestCheckResourceAttr("secobserve_api_configuration.test", "api_key", "secret-key"),
					resource.TestCheckResourceAttr("secobserve_api_configuration.test", "verify_ssl", "false"),
				),
			},
			{
				ResourceName:      "secobserve_api_configuration.test",
				ImportStateIdFunc: apiConfigurationImportID,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// automatic_import_branch has to belong to the same product; SecObserve
// validates this server-side and returns a 400 if it does not. The provider
// does not duplicate that check client-side, so this confirms the server's
// error surfaces as a clear apply-time failure rather than something opaque.
func TestAccAPIConfigurationBranchMustBelongToProduct(t *testing.T) {
	productA := acceptanceName("api-config-branch-a")
	productB := acceptanceName("api-config-branch-b")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "secobserve_product" "a" { name = %q }
resource "secobserve_product" "b" { name = %q }

resource "secobserve_branch" "on_b" {
  product = secobserve_product.b.id
  name    = "main"
}

data "secobserve_parser" "trivy" {
  name = "Trivy Operator Prometheus"
}

resource "secobserve_api_configuration" "test" {
  product                 = secobserve_product.a.id
  name                    = "mismatched-branch"
  parser                  = data.secobserve_parser.trivy.id
  base_url                = "https://trivy.example.com"
  automatic_import_branch = secobserve_branch.on_b.id
}`, productA, productB),
				// Terraform CLI hard-wraps diagnostic text at the terminal
				// width, so a literal space here can land exactly on a wrap
				// boundary; \s+ matches the wrap's real newline too.
				ExpectError: regexp.MustCompile(`(?s)Branch\s+does\s+not\s+belong\s+to\s+the\s+same\s+product`),
			},
		},
	})
}

func TestAccAPIConfigurationBasicAuthPasswordRoundTrips(t *testing.T) {
	product := acceptanceName("api-config-basic-auth")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "secobserve_product" "test" { name = %q }

data "secobserve_parser" "trivy" {
  name = "Trivy Operator Prometheus"
}

resource "secobserve_api_configuration" "test" {
  product             = secobserve_product.test.id
  name                = "basic-auth"
  parser              = data.secobserve_parser.trivy.id
  base_url            = "https://trivy.example.com"
  basic_auth_enabled  = true
  basic_auth_username = "scanner"
  basic_auth_password = "hunter2"
}`, product),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.TestCheckResourceAttr(
					"secobserve_api_configuration.test", "basic_auth_password", "hunter2"),
			},
		},
	})
}

// test_connection is write-only: setting it must not appear as drift on the
// following plan, since it is never returned by the API and the resource
// never writes it into state.
func TestAccAPIConfigurationTestConnectionCausesNoDrift(t *testing.T) {
	product := acceptanceName("api-config-test-conn")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "secobserve_product" "test" { name = %q }

data "secobserve_parser" "trivy" {
  name = "Trivy Operator Prometheus"
}

resource "secobserve_api_configuration" "test" {
  product         = secobserve_product.test.id
  name            = "no-drift"
  parser          = data.secobserve_parser.trivy.id
  base_url        = "https://trivy.example.com"
  test_connection = false
}`, product),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

// apiConfigurationImportID builds the "<product id>/<configuration name>"
// import id from state, since the product's id is only known post-apply.
func apiConfigurationImportID(s *terraform.State) (string, error) {
	product, ok := s.RootModule().Resources["secobserve_product.test"]
	if !ok {
		return "", fmt.Errorf("secobserve_product.test not found in state")
	}
	config, ok := s.RootModule().Resources["secobserve_api_configuration.test"]
	if !ok {
		return "", fmt.Errorf("secobserve_api_configuration.test not found in state")
	}
	return fmt.Sprintf("%s/%s", product.Primary.ID, config.Primary.Attributes["name"]), nil
}
