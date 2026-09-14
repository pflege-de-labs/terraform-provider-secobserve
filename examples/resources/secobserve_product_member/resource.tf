data "secobserve_user" "alice" {
  username = "alice"
}

resource "secobserve_product_member" "alice" {
  product = secobserve_product.checkout.id
  user    = data.secobserve_user.alice.id
  role    = "Maintainer"
}

# Note that whoever creates a product is already an Owner member of it. Since
# Terraform creates products as the provider's own identity, declaring a
# membership for that same user fails as a duplicate.
