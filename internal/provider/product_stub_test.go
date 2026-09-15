package provider_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// These tests run against the in-process stub rather than a real SecObserve,
// so they execute in CI without a container. They target exactly the
// behaviours that would otherwise show up as permanent drift.

// The security_gate block is never populated from the response -- state is
// carried from the plan, unlike every other computed value in this provider.
// Omitting the block entirely means inherit: no block in state at all,
// regardless of whether the gate ends up off (nothing to fill) or on (server
// fills real defaults, but this provider deliberately doesn't surface them;
// see security_gate.go).
func TestProductSecurityGateBlockOmittedMeansInherit(t *testing.T) {
	newStubSecObserve(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "secobserve_product" "test" {
  name = "gate-inherit"
}`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.TestCheckNoResourceAttr("secobserve_product.test", "security_gate.%"),
			},
		},
	})
}

// Writing the block with active = false and nothing else is the explicit way
// to switch the gate off instead of inheriting.
func TestProductSecurityGateExplicitlyDisabled(t *testing.T) {
	newStubSecObserve(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "secobserve_product" "test" {
  name = "gate-disabled"
  security_gate {
    active = false
  }
}`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("secobserve_product.test", "security_gate.active", "false"),
					resource.TestCheckNoResourceAttr("secobserve_product.test", "security_gate.threshold_medium"),
				),
			},
		},
	})
}

// Writing the block with a threshold and no active field implies activating
// the gate. The threshold is echoed back in state unchanged, and a repeat
// apply stays empty, even though the live API's response for the other five,
// unset thresholds differs (the stub fills them per thresholdDefaults) --
// proving state comes from the plan, not the response, for this block.
func TestProductSecurityGateThresholdImpliesActive(t *testing.T) {
	newStubSecObserve(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "secobserve_product" "test" {
  name = "gate-threshold"
  security_gate {
    threshold_medium = 7
  }
}`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("secobserve_product.test", "security_gate.threshold_medium", "7"),
					resource.TestCheckNoResourceAttr("secobserve_product.test", "security_gate.active"),
					resource.TestCheckNoResourceAttr("secobserve_product.test", "security_gate.threshold_low"),
				),
			},
		},
	})
}

// A threshold of 0 is not representable: the fill-in logic tests truthiness,
// so SecObserve replaces it with the instance default. Terraform forbids an
// applied value that differs from the planned one, so accepting the 0 would
// fail the apply with "provider produced inconsistent result". Rejecting it at
// plan time is both earlier and explicable.
func TestProductZeroThresholdRejected(t *testing.T) {
	newStubSecObserve(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "secobserve_product" "test" {
  name = "zero-threshold"
  security_gate {
    threshold_medium = 0
  }
}`,
				ExpectError: regexp.MustCompile(`threshold of 0 cannot be stored`),
			},
		},
	})
}

// An explicit threshold alongside active = false is likewise unrepresentable:
// SecObserve force-clears every threshold whenever the gate is off, so an
// explicit value here would produce the same "provider produced inconsistent
// result" apply failure as the 0 case above.
func TestProductThresholdRejectedWhileGateInactive(t *testing.T) {
	newStubSecObserve(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "secobserve_product" "test" {
  name = "inactive-threshold"
  security_gate {
    active            = false
    threshold_medium  = 7
  }
}`,
				ExpectError: regexp.MustCompile(`Cannot be set while the security gate is inactive`),
			},
		},
	})
}

// Same failure mode for branch housekeeping: a repository_branch_housekeeping
// block with active = false alongside keep_inactive_days/exempt_branches is
// rejected at plan time instead of crashing the apply. This is the exact
// combination reported against a live product_group: keep_inactive_days and
// exempt_branches were left over in config from before housekeeping was
// disabled.
func TestProductHousekeepingFieldsRejectedWhileInactive(t *testing.T) {
	newStubSecObserve(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "secobserve_product" "test" {
  name = "inactive-housekeeping"
  repository_branch_housekeeping {
    active              = false
    keep_inactive_days  = 60
    exempt_branches     = "^(main|release/.*)$"
  }
}`,
				ExpectError: regexp.MustCompile(`Cannot be set while housekeeping is inactive`),
			},
		},
	})
}

// Toggling the repository_branch_housekeeping block between omitted
// (inherit), present-and-empty (implies active) and omitted again has to
// stay a clean, empty-plan no-op throughout -- this is the direct proof the
// refactor removes the "provider produced inconsistent result" bug class
// reported against a live product_group, rather than just renaming the
// attributes: there is no sibling-aware plan modifier left to get this
// wrong, because there's nothing computed to carry forward.
func TestProductHousekeepingBlockPresenceFlip(t *testing.T) {
	newStubSecObserve(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "secobserve_product" "test" {
  name = "housekeeping-flip"
}`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config: `
resource "secobserve_product" "test" {
  name = "housekeeping-flip"
  repository_branch_housekeeping {}
}`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config: `
resource "secobserve_product" "test" {
  name = "housekeeping-flip"
}`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

// Same proof for the security gate.
func TestProductGateBlockPresenceFlip(t *testing.T) {
	newStubSecObserve(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "secobserve_product" "test" {
  name = "gate-flip"
}`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config: `
resource "secobserve_product" "test" {
  name = "gate-flip"
  security_gate {}
}`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config: `
resource "secobserve_product" "test" {
  name = "gate-flip"
}`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

// Omitting security_gate/repository_branch_housekeeping entirely must send
// null so SecObserve keeps inheriting, rather than sending false and pinning
// the value.
func TestProductTriStateStaysNull(t *testing.T) {
	newStubSecObserve(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `resource "secobserve_product" "test" { name = "inherit" }`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("secobserve_product.test", "security_gate.%"),
					resource.TestCheckNoResourceAttr("secobserve_product.test", "repository_branch_housekeeping.%"),
					resource.TestCheckNoResourceAttr("secobserve_product.test", "risk_acceptance_expiry_active"),
					// Defaults, in contrast, are known and sent.
					resource.TestCheckResourceAttr("secobserve_product.test", "apply_general_rules", "true"),
					resource.TestCheckResourceAttr("secobserve_product.test", "osv_enabled", "true"),
					resource.TestCheckResourceAttr("secobserve_product.test", "description", ""),
				),
			},
		},
	})
}

// SecObserve fills in the GitHub API URL when the base URL is omitted.
func TestProductGitHubBaseURLDefault(t *testing.T) {
	newStubSecObserve(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "secobserve_product" "test" {
  name                     = "github"
  issue_tracker_type       = "GitHub"
  issue_tracker_api_key    = "secret"
  issue_tracker_project_id = "org/repo"
}`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.TestCheckResourceAttr(
					"secobserve_product.test", "issue_tracker_base_url", "https://api.github.com"),
			},
		},
	})
}

// An empty propagation list would be stored as null and diff forever, so it is
// rejected at plan time instead.
func TestProductEmptyPropagateBranchesRejected(t *testing.T) {
	newStubSecObserve(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "secobserve_product" "test" {
  name               = "empty-propagation"
  propagate_branches = []
}`,
				ExpectError: regexp.MustCompile(`Empty propagate_branches list`),
			},
		},
	})
}

// The issue tracker all-or-none rule is mirrored client-side so it fails at
// plan time rather than as a 400 mid-apply.
func TestProductIncompleteIssueTrackerRejected(t *testing.T) {
	newStubSecObserve(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "secobserve_product" "test" {
  name               = "half-tracker"
  issue_tracker_type = "GitLab"
}`,
				ExpectError: regexp.MustCompile(`Incomplete issue tracker configuration`),
			},
		},
	})
}

func TestProductJiraRequiresExtraFields(t *testing.T) {
	newStubSecObserve(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "secobserve_product" "test" {
  name                     = "jira"
  issue_tracker_type       = "Jira"
  issue_tracker_base_url   = "https://jira.example.com"
  issue_tracker_api_key    = "secret"
  issue_tracker_project_id = "SEC"
}`,
				ExpectError: regexp.MustCompile(`issue_tracker_username[\s\S]*required for Jira`),
			},
		},
	})
}

// Jira-only fields are rejected for the other tracker types.
func TestProductJiraFieldsRejectedForOtherTypes(t *testing.T) {
	newStubSecObserve(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "secobserve_product" "test" {
  name                     = "gitlab"
  issue_tracker_type       = "GitLab"
  issue_tracker_base_url   = "https://gitlab.example.com"
  issue_tracker_api_key    = "secret"
  issue_tracker_project_id = "42"
  issue_tracker_username   = "bot"
}`,
				ExpectError: regexp.MustCompile(`only valid for Jira`),
			},
		},
	})
}

func TestProductOSVReleaseNeedsDistribution(t *testing.T) {
	newStubSecObserve(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "secobserve_product" "test" {
  name              = "osv"
  osv_linux_release = "22.04"
}`,
				ExpectError: regexp.MustCompile(`osv_linux_release needs a distribution`),
			},
		},
	})
}

// Deleting a product requires its exact name as confirmation; the stub rejects
// a mismatch the way SecObserve does, so a clean destroy proves the provider
// passes it.
func TestProductDeleteSendsNameConfirmation(t *testing.T) {
	newStubSecObserve(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// A name with surrounding whitespace still has to match exactly.
				Config: `resource "secobserve_product" "test" { name = " spaced name " }`,
				Check: resource.TestCheckResourceAttr(
					"secobserve_product.test", "name", " spaced name "),
			},
		},
	})
}

// Renaming has to update the resource rather than replace it, and the
// subsequent destroy has to confirm with the new name.
func TestProductRenameThenDestroy(t *testing.T) {
	newStubSecObserve(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `resource "secobserve_product" "test" { name = "before" }`,
			},
			{
				Config: `resource "secobserve_product" "test" { name = "after" }`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("secobserve_product.test", plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.TestCheckResourceAttr("secobserve_product.test", "name", "after"),
			},
		},
	})
}
