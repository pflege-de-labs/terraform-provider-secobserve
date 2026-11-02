# Emits import blocks and configuration for the rules of one product.
list "secobserve_product_rule" "all" {
  provider = secobserve

  config {
    # A .tfquery.hcl cannot reference resources; use the product id.
    product = 12
  }
}
