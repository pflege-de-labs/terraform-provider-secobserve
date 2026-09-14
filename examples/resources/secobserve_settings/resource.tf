# Every attribute is optional and defaults to SecObserve's own default, so
# only override what actually needs to differ from a fresh install. There is
# exactly one instance of this resource per SecObserve backend.
resource "secobserve_settings" "this" {
  base_url_frontend = "https://secobserve.example.com"

  security_gate_threshold_critical = 0
  security_gate_threshold_high     = 5

  branch_housekeeping_active             = true
  branch_housekeeping_keep_inactive_days = 60

  feature_general_rules_need_approval = true
  risk_acceptance_expiry_days         = 90

  password_validator_minimum_length = 12
}
