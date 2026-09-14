resource "secobserve_general_rule" "ignore_dev_dependencies" {
  name        = "Ignore dev dependencies"
  description = "Dev-only dependencies never ship, so resolve findings on them automatically"
  title       = "^dev-.*"
  new_status  = "Resolved"
}

# Rego rules run a custom policy module instead of a field match.
resource "secobserve_general_rule" "reject_critical_without_fix" {
  name        = "Reject unresolved critical findings"
  description = "Custom policy: block anything critical with no available fix"
  type        = "Rego"
  rego_module = file("${path.module}/rego/reject_critical_without_fix.rego")
}
