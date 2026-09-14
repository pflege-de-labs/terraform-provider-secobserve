data "secobserve_authorization_group" "platform_team" {
  name = "Platform Team"
}

resource "secobserve_product_authorization_group_member" "platform_team" {
  product             = secobserve_product.checkout.id
  authorization_group = data.secobserve_authorization_group.platform_team.id
  role                = "Writer"
}

# A group designated as an assessment approver must already hold at least the
# Writer role, so reference the membership rather than the raw id to get the
# ordering right.
resource "secobserve_product" "with_approvers" {
  name = "checkout-service-approvals"

  assessment_approver_authorization_groups = [
    secobserve_product_authorization_group_member.platform_team.authorization_group,
  ]
}
