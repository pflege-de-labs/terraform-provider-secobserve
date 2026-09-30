data "secobserve_authorization_group" "platform_team" {
  name = "Platform Team"
}

resource "secobserve_product_authorization_group_member" "platform_team" {
  product             = secobserve_product.checkout.id
  authorization_group = data.secobserve_authorization_group.platform_team.id
  role                = "Writer"
}

# A group designated as an assessment approver must already hold at least the
# Writer role on the product or its product group. A new product has no
# memberships yet, so grant the role on the product group and reference the
# membership rather than the raw id to get the ordering right.
resource "secobserve_product_authorization_group_member" "platform_team_payments" {
  product             = secobserve_product_group.payments.id
  authorization_group = data.secobserve_authorization_group.platform_team.id
  role                = "Writer"
}

resource "secobserve_product" "with_approvers" {
  name          = "checkout-service-approvals"
  product_group = secobserve_product_group.payments.id

  assessment_approver_authorization_groups = [
    secobserve_product_authorization_group_member.platform_team_payments.authorization_group,
  ]
}
