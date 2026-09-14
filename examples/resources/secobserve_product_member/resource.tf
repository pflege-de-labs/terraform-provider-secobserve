# Users are looked up by id. The secobserve_user data source arrives with the
# access-control resources; until then, take the id from the SecObserve UI or
# from GET /api/users/.
variable "alice_user_id" {
  type = number
}

resource "secobserve_product_member" "alice" {
  product = secobserve_product.checkout.id
  user    = var.alice_user_id
  role    = "Maintainer"
}

# Note that whoever creates a product is already an Owner member of it. Since
# Terraform creates products as the provider's own identity, declaring a
# membership for that same user fails as a duplicate.
