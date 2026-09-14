resource "secobserve_authorization_group" "legal" {
  name = "Legal"
}

resource "secobserve_license_policy_authorization_group_member" "legal_team" {
  license_policy      = secobserve_license_policy.default.id
  authorization_group = secobserve_authorization_group.legal.id
  is_manager          = true
}
