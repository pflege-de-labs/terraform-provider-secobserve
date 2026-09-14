data "secobserve_user" "legal" {
  username = "legal-review"
}

resource "secobserve_license_group_member" "legal_review" {
  license_group = secobserve_license_group.permissive.id
  user          = data.secobserve_user.legal.id
  is_manager    = true
}
