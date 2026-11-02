# Emits import blocks and configuration for the API configurations of one product.
list "secobserve_api_configuration" "all" {
  provider = secobserve

  config {
    # A .tfquery.hcl cannot reference resources; use the product id.
    product = 12
  }
}
