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

func TestAccProductList(t *testing.T) {
	name := acceptanceName("product")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		TerraformVersionChecks:   skipBelowQuery,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:            fmt.Sprintf(`resource "secobserve_product" "test" { name = %q }`, name),
				ConfigStateChecks: expectIdentity("secobserve_product.test"),
			},
			{
				Query:             true,
				Config:            listQuery("secobserve_product", "all"),
				QueryResultChecks: expectListed("secobserve_product.all", name),
			},
			{
				ResourceName: "secobserve_product.test",
				ImportState:  true,
				// Plannable import asserts a no-op plan, stronger than ImportStateVerify.
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
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
