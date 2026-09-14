data "secobserve_parser" "trivy" {
  name = "Trivy Operator Prometheus"
}

resource "secobserve_api_configuration" "trivy_import" {
  product  = secobserve_product.checkout.id
  name     = "Trivy scan import"
  parser   = data.secobserve_parser.trivy.id
  base_url = "https://trivy.example.com"
  api_key  = var.trivy_api_key

  automatic_import_enabled = true
  automatic_import_branch  = secobserve_branch.main.id
}
