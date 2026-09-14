data "secobserve_user" "legal" {
  username = "legal-review"
}

resource "secobserve_license_policy_member" "legal_review" {
  license_policy = secobserve_license_policy.default.id
  user           = data.secobserve_user.legal.id
  is_manager     = true
}
