# Reference a product managed elsewhere, and read the runtime information the
# secobserve_product resource deliberately does not carry.
data "secobserve_product" "checkout" {
  name = "checkout-service"
}

output "open_critical_findings" {
  value = data.secobserve_product.checkout.active_critical_observation_count
}

output "security_gate_passed" {
  value = data.secobserve_product.checkout.security_gate_passed
}
