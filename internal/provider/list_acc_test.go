package provider_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/querycheck/queryfilter"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
)

// terraform query needs 1.14; resource identity alone needs 1.12.
var (
	skipBelowQuery    = []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_14_0)}
	skipBelowIdentity = []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_12_0)}
)

// listQuery is a .tfquery.hcl body listing every instance of a resource type,
// with full resource objects so the conversion generated config uses runs too.
func listQuery(resourceType, name string) string {
	return fmt.Sprintf(`
provider "secobserve" {}

list %q %q {
  provider         = secobserve
  include_resource = true
}`, resourceType, name)
}

// expectListed asserts that a list block returned an object with this display
// name and, if given, these attribute values in its resource object.
func expectListed(listAddress, displayName string, values ...querycheck.KnownValueCheck) []querycheck.QueryResultCheck {
	byName := queryfilter.ByDisplayName(knownvalue.StringExact(displayName))
	checks := []querycheck.QueryResultCheck{
		querycheck.ExpectLengthAtLeast(listAddress, 1),
		querycheck.ExpectResourceDisplayName(listAddress, byName, knownvalue.StringExact(displayName)),
	}
	if len(values) > 0 {
		checks = append(checks, querycheck.ExpectResourceKnownValues(listAddress, byName, values))
	}
	return checks
}

// setOfSize checks a set attribute's length in a listed resource object.
func setOfSize(attribute string, size int) querycheck.KnownValueCheck {
	return querycheck.KnownValueCheck{Path: tfjsonpath.New(attribute), KnownValue: knownvalue.SetSizeExact(size)}
}

func expectIdentity(address string) []statecheck.StateCheck {
	return []statecheck.StateCheck{
		statecheck.ExpectIdentity(address, map[string]knownvalue.Check{"id": knownvalue.NotNull()}),
	}
}

// testAccList creates one object from config (whose resource is named
// "test"), lists it with terraform query, then re-imports it by identity.
func testAccList(t *testing.T, resourceType, config, displayName string, values ...querycheck.KnownValueCheck) {
	t.Helper()
	address := resourceType + ".test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		TerraformVersionChecks:   skipBelowQuery,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:            config,
				ConfigStateChecks: expectIdentity(address),
			},
			{
				Query:             true,
				Config:            listQuery(resourceType, "all"),
				QueryResultChecks: expectListed(resourceType+".all", displayName, values...),
			},
			{
				ResourceName: address,
				ImportState:  true,
				// Plannable import asserts a no-op plan, stronger than ImportStateVerify.
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

// Approvers are missing from SecObserve's list serializer, so listing must
// read each product's detail or generated config would clear them.
func TestAccProductList(t *testing.T) {
	name := acceptanceName("product")
	testAccList(t, "secobserve_product", fmt.Sprintf(`
resource "secobserve_user" "approver" { username = %q }

resource "secobserve_product" "test" {
  name                 = %q
  assessment_approvers = [secobserve_user.approver.id]
}`, acceptanceName("approver"), name), name, setOfSize("assessment_approvers", 1))
}

func TestAccProductGroupList(t *testing.T) {
	name := acceptanceName("group")
	testAccList(t, "secobserve_product_group", fmt.Sprintf(`
resource "secobserve_user" "approver" { username = %q }

resource "secobserve_product_group" "test" {
  name                 = %q
  assessment_approvers = [secobserve_user.approver.id]
}`, acceptanceName("approver"), name), name, setOfSize("assessment_approvers", 1))
}

// State written by a provider without identity support must refresh cleanly.
func TestAccProductIdentityUpgrade(t *testing.T) {
	config := fmt.Sprintf(`resource "secobserve_product" "test" { name = %q }`, acceptanceName("product"))

	resource.Test(t, resource.TestCase{
		PreCheck:               func() { testAccPreCheck(t) },
		TerraformVersionChecks: skipBelowIdentity,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"secobserve": {Source: "pflege-de-labs/secobserve", VersionConstraint: "0.2.0"},
				},
				Config: config,
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				ConfigStateChecks: expectIdentity("secobserve_product.test"),
			},
		},
	})
}

func TestAccUserList(t *testing.T) {
	name := acceptanceName("user")
	testAccList(t, "secobserve_user",
		fmt.Sprintf(`resource "secobserve_user" "test" { username = %q }`, name), name)
}

func TestAccAuthorizationGroupList(t *testing.T) {
	name := acceptanceName("authz")
	testAccList(t, "secobserve_authorization_group",
		fmt.Sprintf(`resource "secobserve_authorization_group" "test" { name = %q }`, name), name)
}

func TestAccGeneralRuleList(t *testing.T) {
	name := acceptanceName("rule")
	testAccList(t, "secobserve_general_rule", fmt.Sprintf(`
resource "secobserve_general_rule" "test" {
  name        = %q
  description = "ignore dev dependencies"
  title       = "^dev-.*"
  new_status  = "Resolved"
}`, name), name)
}

// The list serializer carries the raw ignore_component_types string only.
func TestAccLicensePolicyList(t *testing.T) {
	name := acceptanceName("policy")
	testAccList(t, "secobserve_license_policy", fmt.Sprintf(`
resource "secobserve_license_policy" "test" {
  name                       = %q
  ignore_component_type_list = ["docker", "npm"]
}`, name), name, setOfSize("ignore_component_type_list", 2))
}

// License groups need one extra request per group for their licenses, so this
// also checks that full resource objects carry them.
func TestAccLicenseGroupList(t *testing.T) {
	name := acceptanceName("license-group")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		TerraformVersionChecks:   skipBelowQuery,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
data "secobserve_license" "mit" { spdx_id = "MIT" }
data "secobserve_license" "apache" { spdx_id = "Apache-2.0" }

resource "secobserve_license_group" "test" {
  name     = %q
  licenses = [data.secobserve_license.mit.id, data.secobserve_license.apache.id]
}`, name),
				ConfigStateChecks: expectIdentity("secobserve_license_group.test"),
			},
			{
				Query: true,
				Config: `
provider "secobserve" {}

list "secobserve_license_group" "all" {
  provider         = secobserve
  include_resource = true
}`,
				QueryResultChecks: append(expectListed("secobserve_license_group.all", name),
					querycheck.ExpectResourceKnownValues("secobserve_license_group.all",
						queryfilter.ByDisplayName(knownvalue.StringExact(name)),
						[]querycheck.KnownValueCheck{{
							Path:       tfjsonpath.New("licenses"),
							KnownValue: knownvalue.SetSizeExact(2),
						}})),
			},
			{
				ResourceName:    "secobserve_license_group.test",
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

// testAccOutOfBandProduct creates a parent product through the API client:
// a .tfquery.hcl cannot reference resources, so the product-scoped list tests
// need its id before the test case starts.
func testAccOutOfBandProduct(t *testing.T) client.Product {
	t.Helper()
	if os.Getenv("TF_ACC") == "" {
		t.Skip("acceptance test; set TF_ACC=1")
	}
	testAccPreCheck(t)

	apiClient, err := client.New(client.Config{
		BaseURL:  os.Getenv("SECOBSERVE_BASE_URL"),
		APIToken: os.Getenv("SECOBSERVE_API_TOKEN"),
		Insecure: os.Getenv("SECOBSERVE_INSECURE") == "true",
	})
	if err != nil {
		t.Fatalf("client.New: %v", err)
	}
	// Name only: ProductRequest always sends every field, several of which reject null.
	var product client.Product
	err = apiClient.Post(context.Background(), "api/products/", map[string]string{"name": acceptanceName("parent")}, &product)
	if err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}
	t.Cleanup(func() {
		if err := apiClient.DeleteProduct(context.Background(), product.ID, product.Name); err != nil && !client.NotFound(err) {
			t.Errorf("DeleteProduct: %v", err)
		}
	})
	return product
}

// testAccProductScopedList is testAccList for resources listed per product.
func testAccProductScopedList(t *testing.T, resourceType, config, displayName string, productID int64) {
	t.Helper()
	address := resourceType + ".test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		TerraformVersionChecks:   skipBelowQuery,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:            config,
				ConfigStateChecks: expectIdentity(address),
			},
			{
				Query: true,
				Config: fmt.Sprintf(`
provider "secobserve" {}

list %q "all" {
  provider         = secobserve
  include_resource = true
  config {
    product = %d
  }
}`, resourceType, productID),
				QueryResultChecks: expectListed(resourceType+".all", displayName),
			},
			{
				ResourceName:    address,
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

func TestAccBranchList(t *testing.T) {
	product := testAccOutOfBandProduct(t)
	testAccProductScopedList(t, "secobserve_branch", fmt.Sprintf(`
resource "secobserve_branch" "test" {
  product = %d
  name    = "main"
}`, product.ID), "main", product.ID)
}

func TestAccServiceList(t *testing.T) {
	product := testAccOutOfBandProduct(t)
	testAccProductScopedList(t, "secobserve_service", fmt.Sprintf(`
resource "secobserve_service" "test" {
  product = %d
  name    = "api"
}`, product.ID), "api", product.ID)
}

func TestAccAPIConfigurationList(t *testing.T) {
	product := testAccOutOfBandProduct(t)
	name := acceptanceName("apiconf")
	testAccProductScopedList(t, "secobserve_api_configuration", fmt.Sprintf(`
data "secobserve_parser" "trivy" {
  name = "Trivy Operator Prometheus"
}

resource "secobserve_api_configuration" "test" {
  product  = %d
  name     = %q
  parser   = data.secobserve_parser.trivy.id
  base_url = "https://trivy.example.com"
  api_key  = "secret-key"
}`, product.ID, name), name, product.ID)
}

func TestAccProductRuleList(t *testing.T) {
	product := testAccOutOfBandProduct(t)
	name := acceptanceName("product-rule")
	testAccProductScopedList(t, "secobserve_product_rule", fmt.Sprintf(`
resource "secobserve_product_rule" "test" {
  product     = %d
  name        = %q
  description = "resolve false positives from this scanner"
  new_status  = "False positive"
}`, product.ID, name), name, product.ID)
}

// A product id that does not exist lists nothing rather than failing.
func TestAccBranchListUnknownProduct(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		TerraformVersionChecks:   skipBelowQuery,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// A query step needs a prior config step; the provider block lives in the query file.
				Config: `locals { unused = true }`,
			},
			{
				Query: true,
				Config: `
provider "secobserve" {}

list "secobserve_branch" "none" {
  provider = secobserve
  config {
    product = 999999999
  }
}`,
				QueryResultChecks: []querycheck.QueryResultCheck{querycheck.ExpectLength("secobserve_branch.none", 0)},
			},
		},
	})
}
