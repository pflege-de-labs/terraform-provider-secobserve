# Local user, not OIDC-provisioned. Has no usable password until one is set
# out of band -- see the resource documentation.
resource "secobserve_user" "ci_bot" {
  username     = "ci-bot"
  first_name   = "CI"
  last_name    = "Bot"
  email        = "ci-bot@example.com"
  is_active    = true
  is_superuser = false
  is_external  = false
}

# Referencing an existing user -- OIDC-provisioned or not -- without
# Terraform claiming ownership of it.
data "secobserve_user" "alice" {
  username = "alice"
}
