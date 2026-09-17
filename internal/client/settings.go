package client

import "context"

// Settings is both the read and write representation of the SecObserve
// instance settings singleton.
//
// Used for both GET and PATCH /api/settings/1/. Every field maps 1:1 to a
// model column (commons/models.py Settings) except
// ObservationTitleNotificationStatusList, which is the serializer's list
// alias for the comma-joined observation_title_notification_statuses column
// (commons/api/serializers.py) -- that column itself is excluded from the
// wire representation entirely and must never be referenced directly.
//
// All fields but one have either a default or blank=True on the model, so
// DRF treats them as optional on write; an absent key on PATCH leaves the
// current value unchanged. ObservationTitleNotificationMinPriority is the one
// genuinely nullable field (IntegerField(null=True), no default) and is a
// pointer accordingly; everything else is a plain value.
type Settings struct {
	SecurityGateActive            bool  `json:"security_gate_active"`
	SecurityGateThresholdCritical int64 `json:"security_gate_threshold_critical"`
	SecurityGateThresholdHigh     int64 `json:"security_gate_threshold_high"`
	SecurityGateThresholdMedium   int64 `json:"security_gate_threshold_medium"`
	SecurityGateThresholdLow      int64 `json:"security_gate_threshold_low"`
	SecurityGateThresholdNone     int64 `json:"security_gate_threshold_none"`
	SecurityGateThresholdUnknown  int64 `json:"security_gate_threshold_unknown"`

	JWTValidityDurationUser      int64  `json:"jwt_validity_duration_user"`
	JWTValidityDurationSuperuser int64  `json:"jwt_validity_duration_superuser"`
	InternalUsers                string `json:"internal_users"`

	// BaseURLFrontend is a pointer, unlike every other string setting, because
	// unlike them it has no blank=True on the model
	// (commons/models.py:69-73): the API rejects an explicit "" with "This
	// field may not be blank." omitempty is what makes "leave it unset"
	// actually omit the key instead of sending the empty string; nil never
	// legitimately reaches the wire otherwise, since the schema validator
	// requires a non-empty value when this is set at all.
	BaseURLFrontend *string `json:"base_url_frontend,omitempty"`

	ExceptionMSTeamsWebhook string `json:"exception_ms_teams_webhook"`
	ExceptionSlackWebhook   string `json:"exception_slack_webhook"`
	ExceptionRateLimit      int64  `json:"exception_rate_limit"`
	EmailFrom               string `json:"email_from"`
	ExceptionEmailTo        string `json:"exception_email_to"`

	ObservationTitleNotificationMSTeamsWebhook string   `json:"observation_title_notification_ms_teams_webhook"`
	ObservationTitleNotificationSlackWebhook   string   `json:"observation_title_notification_slack_webhook"`
	ObservationTitleNotificationEmailTo        string   `json:"observation_title_notification_email_to"`
	ObservationTitleNotificationMinSeverity    string   `json:"observation_title_notification_min_severity"`
	ObservationTitleNotificationStatusList     []string `json:"observation_title_notification_status_list"`
	ObservationTitleNotificationMinPriority    *int64   `json:"observation_title_notification_min_priority"`
	ObservationTitleNotificationParserType     string   `json:"observation_title_notification_parser_type"`

	BackgroundProductMetricsIntervalMinutes int64 `json:"background_product_metrics_interval_minutes"`
	BackgroundEPSSImportCrontabMinute       int64 `json:"background_epss_import_crontab_minute"`
	BackgroundEPSSImportCrontabHour         int64 `json:"background_epss_import_crontab_hour"`

	BranchHousekeepingCrontabMinute    int64  `json:"branch_housekeeping_crontab_minute"`
	BranchHousekeepingCrontabHour      int64  `json:"branch_housekeeping_crontab_hour"`
	BranchHousekeepingActive           bool   `json:"branch_housekeeping_active"`
	BranchHousekeepingKeepInactiveDays int64  `json:"branch_housekeeping_keep_inactive_days"`
	BranchHousekeepingExemptBranches   string `json:"branch_housekeeping_exempt_branches"`

	FeatureVEX            bool   `json:"feature_vex"`
	VEXJustificationStyle string `json:"vex_justification_style"`

	FeatureDisableUserLogin         bool `json:"feature_disable_user_login"`
	FeatureGeneralRulesNeedApproval bool `json:"feature_general_rules_need_approval"`

	RiskAcceptanceExpiryDays          int64 `json:"risk_acceptance_expiry_days"`
	RiskAcceptanceExpiryCrontabMinute int64 `json:"risk_acceptance_expiry_crontab_minute"`
	RiskAcceptanceExpiryCrontabHour   int64 `json:"risk_acceptance_expiry_crontab_hour"`

	FeatureAutomaticAPIImport bool  `json:"feature_automatic_api_import"`
	APIImportCrontabMinute    int64 `json:"api_import_crontab_minute"`
	APIImportCrontabHour      int64 `json:"api_import_crontab_hour"`

	PasswordValidatorMinimumLength       int64 `json:"password_validator_minimum_length"`
	PasswordValidatorAttributeSimilarity bool  `json:"password_validator_attribute_similarity"`
	PasswordValidatorCommonPasswords     bool  `json:"password_validator_common_passwords"`
	PasswordValidatorNotNumeric          bool  `json:"password_validator_not_numeric"`

	// FeatureLicenseManagement is force-enabled server-side whenever
	// FeatureAutomaticOSVScanning is set to true
	// (commons/api/views.py SettingsView.patch), regardless of what is sent
	// for it in the same request.
	FeatureLicenseManagement               bool   `json:"feature_license_management"`
	LicenseImportCrontabMinute             int64  `json:"license_import_crontab_minute"`
	LicenseImportCrontabHour               int64  `json:"license_import_crontab_hour"`
	FeatureAutomaticOSVScanning            bool   `json:"feature_automatic_osv_scanning"`
	FeatureAutomaticVulnerableCodeScanning bool   `json:"feature_automatic_vulnerablecode_scanning"`
	VulnerableCodeBaseURL                  string `json:"vulnerablecode_base_url"`
	// VulnerableCodeAPIKey is encrypted at rest but returned in plaintext to a
	// superuser caller -- unlike the product/api_configuration API keys, it is
	// not stripped from responses.
	VulnerableCodeAPIKey          string `json:"vulnerablecode_api_key"`
	FeatureExploitInformation     bool   `json:"feature_exploit_information"`
	ExploitInformationMaxAgeYears int64  `json:"exploit_information_max_age_years"`
	PeriodicTaskMaxEntries        int64  `json:"periodic_task_max_entries"`

	OIDCClockSkew      int64 `json:"oidc_clock_skew"`
	OIDCStrictAudience bool  `json:"oidc_strict_audience"`

	ObservationCountFromMetrics bool `json:"observation_count_from_metrics"`

	FeatureCrossScannerDeduplication bool `json:"feature_cross_scanner_deduplication"`
	FeatureShowProductHeaderChips    bool `json:"feature_show_product_header_chips"`
}

const settingsPath = "api/settings/1/"

// GetSettings reads the instance settings singleton. Superuser-only.
func (c *Client) GetSettings(ctx context.Context) (Settings, error) {
	var settings Settings
	err := c.Get(ctx, settingsPath, nil, &settings)
	return settings, err
}

// UpdateSettings patches the instance settings singleton with the full
// managed field set. There is no create or delete on this endpoint: the
// singleton always exists (commons/models.py Settings.load() returns a
// default instance even before one has ever been saved).
func (c *Client) UpdateSettings(ctx context.Context, request Settings) (Settings, error) {
	var updated Settings
	err := c.Patch(ctx, settingsPath, request, &updated)
	return updated, err
}
