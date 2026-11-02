# Run `terraform query -generate-config-out=generated.tf` to emit import blocks
# and configuration for every product.
list "secobserve_product" "all" {
  provider = secobserve
}
