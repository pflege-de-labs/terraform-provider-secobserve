package provider_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Parsers are seeded by SecObserve itself at startup, so this exercises the
// whole stack -- auth header, pagination, exact-name lookup -- against a real
// instance without creating anything.
func TestAccParserDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "secobserve_parser" "manual" { name = "Manual" }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.secobserve_parser.manual", "name", "Manual"),
					resource.TestCheckResourceAttr("data.secobserve_parser.manual", "source", "Manual"),
					resource.TestCheckResourceAttrSet("data.secobserve_parser.manual", "id"),
				),
			},
		},
	})
}

func TestAccParserDataSourceNotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      `data "secobserve_parser" "missing" { name = "No Such Parser" }`,
				ExpectError: regexp.MustCompile(`no parser found with name "No Such Parser"`),
			},
		},
	})
}

// The API filters names with icontains. A prefix of a real parser name must
// not silently resolve to it.
func TestAccParserDataSourceRejectsPartialName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      `data "secobserve_parser" "partial" { name = "Manua" }`,
				ExpectError: regexp.MustCompile(`no parser found with name "Manua"`),
			},
		},
	})
}

func TestAccParsersDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "secobserve_parsers" "api" { source = "API" }`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.secobserve_parsers.api", "parsers.#"),
					resource.TestCheckResourceAttr("data.secobserve_parsers.api", "parsers.0.source", "API"),
				),
			},
		},
	})
}
