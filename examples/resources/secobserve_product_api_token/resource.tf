resource "secobserve_product_api_token" "ci" {
  product = secobserve_product.checkout.id
  name    = "github-actions" # at most 32 characters
  role    = "Upload"

  expiration_date = "2027-01-01"
}

# The secret is returned only once, when the token is created, so it lives in
# Terraform state. Any change to this resource replaces the token and issues a
# new secret.
output "ci_token" {
  value     = secobserve_product_api_token.ci.token
  sensitive = true
}
