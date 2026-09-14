# Reference the seeded "Standard" policy without adopting it into Terraform.
data "secobserve_license_policy" "standard" {
  name = "Standard"
}

resource "secobserve_product_group" "internal_tools" {
  name           = "internal-tools"
  license_policy = data.secobserve_license_policy.standard.id
}
