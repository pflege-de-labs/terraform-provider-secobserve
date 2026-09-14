# The ScanCode LicenseDB groups are system-managed and reimported nightly;
# reference them read-only rather than owning them with the resource.
data "secobserve_license_group" "scancode_permissive" {
  name = "Permissive (ScanCode LicenseDB)"
}

output "scancode_permissive_license_ids" {
  value = data.secobserve_license_group.scancode_permissive.licenses
}
