resource "secobserve_branch" "main" {
  product = secobserve_product.checkout.id
  name    = "main"

  # Exactly one branch per product should set this: SecObserve clears the flag
  # on the product's other branches, which Terraform cannot observe. The
  # default branch also cannot be deleted, so move the flag before destroying.
  is_default_branch    = true
  housekeeping_protect = true
}

resource "secobserve_branch" "release" {
  product              = secobserve_product.checkout.id
  name                 = "release/2026.1"
  housekeeping_protect = true
}
