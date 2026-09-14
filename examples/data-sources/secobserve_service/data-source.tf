data "secobserve_service" "api" {
  product = data.secobserve_product.checkout.id
  name    = "api"
}
