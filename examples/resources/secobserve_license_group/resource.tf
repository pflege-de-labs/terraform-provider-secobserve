data "secobserve_license" "mit" {
  spdx_id = "MIT"
}

data "secobserve_license" "apache" {
  spdx_id = "Apache-2.0"
}

resource "secobserve_license_group" "permissive" {
  name        = "Permissive (approved)"
  description = "Licenses pre-approved for use without review"

  licenses = [
    data.secobserve_license.mit.id,
    data.secobserve_license.apache.id,
  ]
}
