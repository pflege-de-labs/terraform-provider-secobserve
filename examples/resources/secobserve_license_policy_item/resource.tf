data "secobserve_license" "gpl" {
  spdx_id = "GPL-3.0-only"
}

resource "secobserve_license_policy_item" "allow_permissive" {
  license_policy    = secobserve_license_policy.default.id
  license_group     = secobserve_license_group.permissive.id
  evaluation_result = "Allowed"
}

resource "secobserve_license_policy_item" "forbid_gpl" {
  license_policy    = secobserve_license_policy.default.id
  license           = data.secobserve_license.gpl.id
  evaluation_result = "Forbidden"
  comment           = "Copyleft, incompatible with our distribution model"
}
