# The documented way to reference a user -- OIDC-provisioned or local --
# without Terraform claiming ownership of it.
data "secobserve_user" "alice" {
  username = "alice"
}

output "alice_id" {
  value = data.secobserve_user.alice.id
}
