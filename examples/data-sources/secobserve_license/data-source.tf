# Resolve an SPDX license id to the numeric id secobserve_license_group and
# secobserve_license_policy_item expect.
data "secobserve_license" "mit" {
  spdx_id = "MIT"
}

output "mit_license_id" {
  value = data.secobserve_license.mit.id
}
