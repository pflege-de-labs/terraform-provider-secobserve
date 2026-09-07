resource "secobserve_service" "api" {
  product = secobserve_product.checkout.id
  name    = "api"
}
