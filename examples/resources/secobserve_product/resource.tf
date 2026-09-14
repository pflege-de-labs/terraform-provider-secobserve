resource "secobserve_product" "checkout" {
  name          = "checkout-service"
  description   = "Customer-facing checkout API"
  product_group = secobserve_product_group.payments.id

  repository_prefix = "https://github.com/example/checkout-service"
  purl              = "pkg:github/example/checkout-service"

  # Leave security_gate_active unset to inherit the product group's setting.
  # Setting it to false would switch the gate off for this product instead.

  # Push findings to GitHub issues. SecObserve requires the type, base URL,
  # API key and project id to be either all set or all empty; for GitHub the
  # base URL may be omitted and is filled in as https://api.github.com.
  issue_tracker_active           = true
  issue_tracker_type             = "GitHub"
  issue_tracker_api_key          = var.github_token
  issue_tracker_project_id       = "example/checkout-service"
  issue_tracker_labels           = "security,dependencies"
  issue_tracker_minimum_severity = "High"

  osv_enabled                    = true
  osv_linux_distribution         = "Debian"
  osv_linux_release              = "12"
  automatic_osv_scanning_enabled = true

  propagate_branches = [
    { propagate_to = "^release/.*$" },
  ]
}
