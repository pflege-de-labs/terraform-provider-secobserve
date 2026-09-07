resource "secobserve_product_group" "payments" {
  name        = "Payments"
  description = "Everything that moves money"

  # Products in this group inherit these unless they set their own.
  security_gate_active             = true
  security_gate_threshold_critical = 1
  security_gate_threshold_high     = 5

  repository_branch_housekeeping_active             = true
  repository_branch_housekeeping_keep_inactive_days = 60
  repository_branch_housekeeping_exempt_branches    = "^(main|release/.*)$"

  notification_slack_webhook            = "https://hooks.slack.com/services/T000/B000/XXXX"
  observation_notification_min_severity = "High"
  observation_notification_status_list  = ["Open", "Affected"]

  assessments_need_approval   = true
  product_rules_need_approval = true
}
