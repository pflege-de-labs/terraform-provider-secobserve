# Emits import blocks and configuration for the branches of one product.
list "secobserve_branch" "all" {
  provider = secobserve

  config {
    # A .tfquery.hcl cannot reference resources; use the product id.
    product = 12
  }
}
