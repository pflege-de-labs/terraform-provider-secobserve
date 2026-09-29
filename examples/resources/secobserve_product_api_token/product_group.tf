# A token scoped to a product group holds its role on every product in the
# group, so one pipeline token can upload to all of them.
resource "secobserve_product_group" "payments" {
  name = "payments"
}

resource "secobserve_product_api_token" "payments_ci" {
  product = secobserve_product_group.payments.id
  name    = "payments-ci"
  role    = "Upload"
}
