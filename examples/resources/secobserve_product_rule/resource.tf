resource "secobserve_product_rule" "suppress_scanner_false_positive" {
  product        = secobserve_product.checkout.id
  name           = "Suppress known scanner false positive"
  description    = "This finding does not apply to how we use the library"
  scanner_prefix = "Trivy"
  new_status     = "False positive"

  new_vex_remediations = [
    { category = "workaround", text = "Not exploitable in our usage" },
  ]
}
