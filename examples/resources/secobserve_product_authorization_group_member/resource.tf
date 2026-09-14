variable "platform_team_group_id" {
  type = number
}

resource "secobserve_product_authorization_group_member" "platform_team" {
  product             = secobserve_product.checkout.id
  authorization_group = var.platform_team_group_id
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
