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

// With the gate off, SecObserve clears every threshold. The provider has to
// take that from the response instead of the plan, or the next plan is dirty.
func TestProductSecurityGateInactiveClearsThresholds(t *testing.T) {
	newStubSecObserve(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "secobserve_product" "test" {
  name                 = "gate-off"
  security_gate_active = false
}`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("secobserve_product.test", "security_gate_active", "false"),
					resource.TestCheckNoResourceAttr("secobserve_product.test", "security_gate_threshold_medium"),
				),
			},
		},
	})
}

// With the gate on and no thresholds given, SecObserve fills them from the
// instance settings.
func TestProductSecurityGateActiveFillsThresholds(t *testing.T) {
	newStubSecObserve(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "secobserve_product" "test" {
  name                 = "gate-on"
  security_gate_active = true
}`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("secobserve_product.test", "security_gate_threshold_medium", "99999"),
					resource.TestCheckResourceAttr("secobserve_product.test", "security_gate_threshold_critical", "0"),
				),
			},
		},
	})
}

// Turning the gate off after it was on clears the thresholds server-side. This
// is the case that proves state is written from the response body: the plan
// still carries the old values at that point.
func TestProductSecurityGateFlipClearsThresholds(t *testing.T) {
	newStubSecObserve(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "secobserve_product" "test" {
  name                           = "gate-flip"
  security_gate_active           = true
  security_gate_threshold_medium = 7
}`,
				Check: resource.TestCheckResourceAttr(
					"secobserve_product.test", "security_gate_threshold_medium", "7"),
			},
			{
				Config: `
resource "secobserve_product" "test" {
  name                 = "gate-flip"
  security_gate_active = false
}`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.TestCheckNoResourceAttr(
					"secobserve_product.test", "security_gate_threshold_medium"),
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
  name                           = "zero-threshold"
  security_gate_active           = true
  security_gate_threshold_medium = 0
}`,
				ExpectError: regexp.MustCompile(`threshold of 0 cannot be stored`),
			},
		},
	})
}

// Flipping the gate while the thresholds are never configured is what the
// sibling-aware plan modifier exists for: the framework would otherwise carry
// the prior state values into the plan and Terraform would reject the server's
// recomputed answer as an inconsistent result.
func TestProductGateFlipWithUnsetThresholds(t *testing.T) {
	newStubSecObserve(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "secobserve_product" "test" {
  name                 = "flip-unset"
  security_gate_active = true
}`,
				Check: resource.TestCheckResourceAttr(
					"secobserve_product.test", "security_gate_threshold_low", "99999"),
			},
			{
				// Off: the server clears all six.
				Config: `
resource "secobserve_product" "test" {
  name                 = "flip-unset"
  security_gate_active = false
}`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.TestCheckNoResourceAttr(
					"secobserve_product.test", "security_gate_threshold_low"),
			},
			{
				// Back on: the server fills all six again.
				Config: `
resource "secobserve_product" "test" {
  name                 = "flip-unset"
  security_gate_active = true
}`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.TestCheckResourceAttr(
					"secobserve_product.test", "security_gate_threshold_low", "99999"),
			},
		},
	})
}

// Leaving a tri-state unset must send null so SecObserve keeps inheriting,
// rather than sending false and pinning the value.
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
					resource.TestCheckNoResourceAttr("secobserve_product.test", "security_gate_active"),
					resource.TestCheckNoResourceAttr("secobserve_product.test", "repository_branch_housekeeping_active"),
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
