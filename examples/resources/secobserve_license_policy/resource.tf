resource "secobserve_license_policy" "default" {
  name        = "Company default"
  description = "Baseline policy applied to every product unless overridden"

  # Container base images are scanned separately; don't flag their licenses here.
  ignore_component_type_list = ["docker"]
}

resource "secobserve_product" "checkout" {
  name           = "checkout-service"
  license_policy = secobserve_license_policy.default.id
}
