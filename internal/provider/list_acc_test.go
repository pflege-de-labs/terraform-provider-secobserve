package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/querycheck/queryfilter"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

// terraform query needs 1.14; resource identity alone needs 1.12.
var (
	skipBelowQuery    = []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_14_0)}
	skipBelowIdentity = []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_12_0)}
)

// listQuery is a .tfquery.hcl body listing every instance of a resource type.
func listQuery(resourceType, name string) string {
	return fmt.Sprintf(`
provider "secobserve" {}

list %q %q {
  provider = secobserve
}`, resourceType, name)
}

// expectListed asserts that a list block returned an object with this display name.
func expectListed(listAddress, displayName string) []querycheck.QueryResultCheck {
	return []querycheck.QueryResultCheck{
		querycheck.ExpectLengthAtLeast(listAddress, 1),
		querycheck.ExpectResourceDisplayName(listAddress,
			queryfilter.ByDisplayName(knownvalue.StringExact(displayName)),
			knownvalue.StringExact(displayName)),
	}
}

func expectIdentity(address string) []statecheck.StateCheck {
	return []statecheck.StateCheck{
		statecheck.ExpectIdentity(address, map[string]knownvalue.Check{"id": knownvalue.NotNull()}),
	}
}

// testAccList creates one object from config (whose resource is named
// "test"), lists it with terraform query, then re-imports it by identity.
func testAccList(t *testing.T, resourceType, config, displayName string) {
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
				QueryResultChecks: expectListed(resourceType+".all", displayName),
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

func TestAccProductList(t *testing.T) {
	name := acceptanceName("product")
	testAccList(t, "secobserve_product",
		fmt.Sprintf(`resource "secobserve_product" "test" { name = %q }`, name), name)
}

func TestAccProductGroupList(t *testing.T) {
	name := acceptanceName("group")
	testAccList(t, "secobserve_product_group",
		fmt.Sprintf(`resource "secobserve_product_group" "test" { name = %q }`, name), name)
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

func TestAccLicensePolicyList(t *testing.T) {
	name := acceptanceName("policy")
	testAccList(t, "secobserve_license_policy",
		fmt.Sprintf(`resource "secobserve_license_policy" "test" { name = %q }`, name), name)
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
