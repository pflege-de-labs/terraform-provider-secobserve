data "secobserve_user" "alice" {
  username = "alice"
}

resource "secobserve_authorization_group_member" "alice" {
  authorization_group = secobserve_authorization_group.platform_team.id
  user                = data.secobserve_user.alice.id
  is_manager          = true
}

# Do not do this for a group with oidc_group set: membership of an
# OIDC-managed group is deleted by SecObserve's login-time group sync,
# regardless of what Terraform declares. See secobserve_authorization_group.
