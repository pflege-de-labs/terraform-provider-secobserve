data "secobserve_product_group" "payments" {
  name = "Payments"
}

output "products_in_group" {
  value = data.secobserve_product_group.payments.products_count
}
