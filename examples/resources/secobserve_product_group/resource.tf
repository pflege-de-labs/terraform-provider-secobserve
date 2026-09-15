resource "secobserve_product_group" "payments" {
  name        = "Payments"
  description = "Everything that moves money"

  # Products in this group inherit these unless they set their own.
  security_gate_active = true
  security_gate = {
    threshold_critical = 1
    threshold_high     = 5
  }

  repository_branch_housekeeping_active = true
  repository_branch_housekeeping = {
    keep_inactive_days = 60
    exempt_branches    = "^(main|release/.*)$"
  }

  notification_slack_webhook            = "https://hooks.slack.com/services/T000/B000/XXXX"
  observation_notification_min_severity = "High"
  observation_notification_status_list  = ["Open", "Affected"]

  assessments_need_approval   = true
  product_rules_need_approval = true
}
