# Emits import blocks and configuration for the services of one product.
list "secobserve_service" "all" {
  provider = secobserve

  config {
    # A .tfquery.hcl cannot reference resources; use the product id.
    product = 12
  }
}
