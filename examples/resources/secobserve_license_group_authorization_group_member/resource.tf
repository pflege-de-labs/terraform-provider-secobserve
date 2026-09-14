resource "secobserve_authorization_group" "legal" {
  name = "Legal"
}

resource "secobserve_license_group_authorization_group_member" "legal_team" {
  license_group       = secobserve_license_group.permissive.id
  authorization_group = secobserve_authorization_group.legal.id
  is_manager          = true
}
