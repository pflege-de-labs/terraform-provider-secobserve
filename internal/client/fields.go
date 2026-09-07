package client

// The field blocks below are shared by products and product groups. They are
// embedded into the request and response types, which encoding/json flattens,
// so each block has exactly one definition and the provider can copy it as a
// unit.
//
// Pointers mean "nullable in the API". Plain strings and bools are fields the
// backend declares blank=True or with a default and which therefore reject a
// JSON null.

// SecurityGateFields controls when a product fails its security gate.
//
// SecObserve rewrites these: with the gate active it fills unset thresholds
// from the instance settings, and with the gate off it clears all of them
// (core/api/serializers_product.py:122-142). The fill test is truthiness, not
// presence, so a submitted 0 is also replaced.
type SecurityGateFields struct {
	SecurityGateActive            *bool  `json:"security_gate_active"`
	SecurityGateThresholdCritical *int64 `json:"security_gate_threshold_critical"`
	SecurityGateThresholdHigh     *int64 `json:"security_gate_threshold_high"`
	SecurityGateThresholdMedium   *int64 `json:"security_gate_threshold_medium"`
	SecurityGateThresholdLow      *int64 `json:"security_gate_threshold_low"`
	SecurityGateThresholdNone     *int64 `json:"security_gate_threshold_none"`
	SecurityGateThresholdUnknown  *int64 `json:"security_gate_threshold_unknown"`
}

// BranchHousekeepingFields controls automatic deletion of inactive branches.
// Subject to the same fill-and-clear behaviour as the security gate.
type BranchHousekeepingFields struct {
	RepositoryBranchHousekeepingActive           *bool  `json:"repository_branch_housekeeping_active"`
	RepositoryBranchHousekeepingKeepInactiveDays *int64 `json:"repository_branch_housekeeping_keep_inactive_days"`
	RepositoryBranchHousekeepingExemptBranches   string `json:"repository_branch_housekeeping_exempt_branches"`
}

// NotificationFields configures where observation notifications go and which
// observations trigger one.
type NotificationFields struct {
	NotificationMSTeamsWebhook string `json:"notification_ms_teams_webhook"`
	NotificationSlackWebhook   string `json:"notification_slack_webhook"`
	NotificationEmailTo        string `json:"notification_email_to"`

	ObservationNotificationMinSeverity string `json:"observation_notification_min_severity"`
	// Serializer-only alias for the comma-joined observation_notification_statuses
	// column (core/api/serializers_product.py:256-270).
	ObservationNotificationStatusList  []string `json:"observation_notification_status_list"`
	ObservationNotificationMinPriority *int64   `json:"observation_notification_min_priority"`
}

// ApprovalFields controls the review and approval workflows.
type ApprovalFields struct {
	AssessmentsNeedApproval  bool `json:"assessments_need_approval"`
	NewObservationsInReview  bool `json:"new_observations_in_review"`
	ProductRulesNeedApproval bool `json:"product_rules_need_approval"`
}

// ApproverFields designates who may approve assessments.
//
// Each authorization group must already hold at least the Writer role on the
// product or its group, or the request is rejected
// (core/api/serializers_product.py:149-189).
type ApproverFields struct {
	AssessmentApprovers                   []int64 `json:"assessment_approvers"`
	AssessmentApproverAuthorizationGroups []int64 `json:"assessment_approver_authorization_groups"`
}

// RiskAcceptanceFields controls expiry of accepted risks.
type RiskAcceptanceFields struct {
	RiskAcceptanceExpiryActive *bool `json:"risk_acceptance_expiry_active"`
	// Zero means accepted risks never expire.
	RiskAcceptanceExpiryDays *int64 `json:"risk_acceptance_expiry_days"`
}

// LicensePolicyFields links a product or group to a license policy.
type LicensePolicyFields struct {
	LicensePolicy *int64 `json:"license_policy"`
}

// PropagateBranch is one branch propagation rule.
type PropagateBranch struct {
	PropagateTo string `json:"propagate_to"`
}

// BranchPropagationFields controls copying observations and assessments
// between branches.
//
// An empty or effectively-empty list is stored as null rather than []
// (core/api/serializers_product.py:660-663).
type BranchPropagationFields struct {
	PropagateBranches               []PropagateBranch `json:"propagate_branches"`
	PropagateBranchesNewAssessment  bool              `json:"propagate_branches_new_assessment"`
	PropagateBranchesNewObservation bool              `json:"propagate_branches_new_observation"`
}

// IssueTrackerFields configures pushing observations into an issue tracker.
//
// SecObserve enforces that type, base URL, API key and project id are either
// all set or all empty (core/api/serializers_product.py:565-574), which is why
// the API key has to be readable rather than write-only.
type IssueTrackerFields struct {
	IssueTrackerActive bool   `json:"issue_tracker_active"`
	IssueTrackerType   string `json:"issue_tracker_type"`
	// Filled in with GitHubDefaultBaseURL when the type is GitHub and this is
	// empty.
	IssueTrackerBaseURL         string `json:"issue_tracker_base_url"`
	IssueTrackerUsername        string `json:"issue_tracker_username"`
	IssueTrackerAPIKey          string `json:"issue_tracker_api_key"`
	IssueTrackerProjectID       string `json:"issue_tracker_project_id"`
	IssueTrackerLabels          string `json:"issue_tracker_labels"`
	IssueTrackerIssueType       string `json:"issue_tracker_issue_type"`
	IssueTrackerStatusClosed    string `json:"issue_tracker_status_closed"`
	IssueTrackerMinimumSeverity string `json:"issue_tracker_minimum_severity"`
}

// ScannerFields configures the scanners SecObserve runs itself.
type ScannerFields struct {
	OSVEnabled           bool   `json:"osv_enabled"`
	OSVLinuxDistribution string `json:"osv_linux_distribution"`
	// Cannot be set without OSVLinuxDistribution.
	OSVLinuxRelease             string `json:"osv_linux_release"`
	AutomaticOSVScanningEnabled bool   `json:"automatic_osv_scanning_enabled"`

	VulnerableCodeEnabled                  bool `json:"vulnerablecode_enabled"`
	AutomaticVulnerableCodeScanningEnabled bool `json:"automatic_vulnerablecode_scanning_enabled"`
}
