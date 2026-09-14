# A group whose membership Terraform manages directly.
resource "secobserve_authorization_group" "platform_team" {
  name = "Platform Team"
}

# A group whose membership is synchronized from an IdP group claim instead.
# Do not create secobserve_authorization_group_member resources for this one
# -- SecObserve's login-time group sync would delete them again.
resource "secobserve_authorization_group" "security_team" {
  name       = "Security Team"
  oidc_group = "security-team"
}
