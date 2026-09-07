# Branch names are unique per product, so both are required.
data "secobserve_branch" "main" {
  product = data.secobserve_product.checkout.id
  name    = "main"
}
