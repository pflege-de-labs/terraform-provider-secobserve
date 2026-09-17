package settings

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tfutil"
)

// model mirrors the resource schema, field for field against client.Settings.
type model struct {
	ID types.Int64 `tfsdk:"id"`

	SecurityGateActive            types.Bool  `tfsdk:"security_gate_active"`
	SecurityGateThresholdCritical types.Int64 `tfsdk:"security_gate_threshold_critical"`
	SecurityGateThresholdHigh     types.Int64 `tfsdk:"security_gate_threshold_high"`
	SecurityGateThresholdMedium   types.Int64 `tfsdk:"security_gate_threshold_medium"`
	SecurityGateThresholdLow      types.Int64 `tfsdk:"security_gate_threshold_low"`
	SecurityGateThresholdNone     types.Int64 `tfsdk:"security_gate_threshold_none"`
	SecurityGateThresholdUnknown  types.Int64 `tfsdk:"security_gate_threshold_unknown"`

	JWTValidityDurationUser      types.Int64  `tfsdk:"jwt_validity_duration_user"`
	JWTValidityDurationSuperuser types.Int64  `tfsdk:"jwt_validity_duration_superuser"`
	InternalUsers                types.String `tfsdk:"internal_users"`

	BaseURLFrontend types.String `tfsdk:"base_url_frontend"`

	ExceptionMSTeamsWebhook types.String `tfsdk:"exception_ms_teams_webhook"`
	ExceptionSlackWebhook   types.String `tfsdk:"exception_slack_webhook"`
	ExceptionRateLimit      types.Int64  `tfsdk:"exception_rate_limit"`
	EmailFrom               types.String `tfsdk:"email_from"`
	ExceptionEmailTo        types.String `tfsdk:"exception_email_to"`

	ObservationTitleNotificationMSTeamsWebhook types.String `tfsdk:"observation_title_notification_ms_teams_webhook"`
	ObservationTitleNotificationSlackWebhook   types.String `tfsdk:"observation_title_notification_slack_webhook"`
	ObservationTitleNotificationEmailTo        types.String `tfsdk:"observation_title_notification_email_to"`
	ObservationTitleNotificationMinSeverity    types.String `tfsdk:"observation_title_notification_min_severity"`
	ObservationTitleNotificationStatusList     types.Set    `tfsdk:"observation_title_notification_status_list"`
	ObservationTitleNotificationMinPriority    types.Int64  `tfsdk:"observation_title_notification_min_priority"`
	ObservationTitleNotificationParserType     types.String `tfsdk:"observation_title_notification_parser_type"`

	BackgroundProductMetricsIntervalMinutes types.Int64 `tfsdk:"background_product_metrics_interval_minutes"`
	BackgroundEPSSImportCrontabMinute       types.Int64 `tfsdk:"background_epss_import_crontab_minute"`
	BackgroundEPSSImportCrontabHour         types.Int64 `tfsdk:"background_epss_import_crontab_hour"`

	BranchHousekeepingCrontabMinute    types.Int64  `tfsdk:"branch_housekeeping_crontab_minute"`
	BranchHousekeepingCrontabHour      types.Int64  `tfsdk:"branch_housekeeping_crontab_hour"`
	BranchHousekeepingActive           types.Bool   `tfsdk:"branch_housekeeping_active"`
	BranchHousekeepingKeepInactiveDays types.Int64  `tfsdk:"branch_housekeeping_keep_inactive_days"`
	BranchHousekeepingExemptBranches   types.String `tfsdk:"branch_housekeeping_exempt_branches"`

	FeatureVEX            types.Bool   `tfsdk:"feature_vex"`
	VEXJustificationStyle types.String `tfsdk:"vex_justification_style"`

	FeatureDisableUserLogin         types.Bool `tfsdk:"feature_disable_user_login"`
	FeatureGeneralRulesNeedApproval types.Bool `tfsdk:"feature_general_rules_need_approval"`

	RiskAcceptanceExpiryDays          types.Int64 `tfsdk:"risk_acceptance_expiry_days"`
	RiskAcceptanceExpiryCrontabMinute types.Int64 `tfsdk:"risk_acceptance_expiry_crontab_minute"`
	RiskAcceptanceExpiryCrontabHour   types.Int64 `tfsdk:"risk_acceptance_expiry_crontab_hour"`

	FeatureAutomaticAPIImport types.Bool  `tfsdk:"feature_automatic_api_import"`
	APIImportCrontabMinute    types.Int64 `tfsdk:"api_import_crontab_minute"`
	APIImportCrontabHour      types.Int64 `tfsdk:"api_import_crontab_hour"`

	PasswordValidatorMinimumLength       types.Int64 `tfsdk:"password_validator_minimum_length"`
	PasswordValidatorAttributeSimilarity types.Bool  `tfsdk:"password_validator_attribute_similarity"`
	PasswordValidatorCommonPasswords     types.Bool  `tfsdk:"password_validator_common_passwords"`
	PasswordValidatorNotNumeric          types.Bool  `tfsdk:"password_validator_not_numeric"`

	FeatureLicenseManagement               types.Bool   `tfsdk:"feature_license_management"`
	LicenseImportCrontabMinute             types.Int64  `tfsdk:"license_import_crontab_minute"`
	LicenseImportCrontabHour               types.Int64  `tfsdk:"license_import_crontab_hour"`
	FeatureAutomaticOSVScanning            types.Bool   `tfsdk:"feature_automatic_osv_scanning"`
	FeatureAutomaticVulnerableCodeScanning types.Bool   `tfsdk:"feature_automatic_vulnerablecode_scanning"`
	VulnerableCodeBaseURL                  types.String `tfsdk:"vulnerablecode_base_url"`
	VulnerableCodeAPIKey                   types.String `tfsdk:"vulnerablecode_api_key"`
	FeatureExploitInformation              types.Bool   `tfsdk:"feature_exploit_information"`
	ExploitInformationMaxAgeYears          types.Int64  `tfsdk:"exploit_information_max_age_years"`
	PeriodicTaskMaxEntries                 types.Int64  `tfsdk:"periodic_task_max_entries"`

	OIDCClockSkew      types.Int64 `tfsdk:"oidc_clock_skew"`
	OIDCStrictAudience types.Bool  `tfsdk:"oidc_strict_audience"`

	ObservationCountFromMetrics types.Bool `tfsdk:"observation_count_from_metrics"`

	FeatureCrossScannerDeduplication types.Bool `tfsdk:"feature_cross_scanner_deduplication"`
	FeatureShowProductHeaderChips    types.Bool `tfsdk:"feature_show_product_header_chips"`
}

func (m model) toRequest(ctx context.Context, diags *diag.Diagnostics) client.Settings {
	return client.Settings{
		SecurityGateActive:            tfutil.BoolValue(m.SecurityGateActive),
		SecurityGateThresholdCritical: m.SecurityGateThresholdCritical.ValueInt64(),
		SecurityGateThresholdHigh:     m.SecurityGateThresholdHigh.ValueInt64(),
		SecurityGateThresholdMedium:   m.SecurityGateThresholdMedium.ValueInt64(),
		SecurityGateThresholdLow:      m.SecurityGateThresholdLow.ValueInt64(),
		SecurityGateThresholdNone:     m.SecurityGateThresholdNone.ValueInt64(),
		SecurityGateThresholdUnknown:  m.SecurityGateThresholdUnknown.ValueInt64(),

		JWTValidityDurationUser:      m.JWTValidityDurationUser.ValueInt64(),
		JWTValidityDurationSuperuser: m.JWTValidityDurationSuperuser.ValueInt64(),
		InternalUsers:                tfutil.StringValue(m.InternalUsers),

		BaseURLFrontend: tfutil.StringPtr(m.BaseURLFrontend),

		ExceptionMSTeamsWebhook: tfutil.StringValue(m.ExceptionMSTeamsWebhook),
		ExceptionSlackWebhook:   tfutil.StringValue(m.ExceptionSlackWebhook),
		ExceptionRateLimit:      m.ExceptionRateLimit.ValueInt64(),
		EmailFrom:               tfutil.StringValue(m.EmailFrom),
		ExceptionEmailTo:        tfutil.StringValue(m.ExceptionEmailTo),

		ObservationTitleNotificationMSTeamsWebhook: tfutil.StringValue(m.ObservationTitleNotificationMSTeamsWebhook),
		ObservationTitleNotificationSlackWebhook:   tfutil.StringValue(m.ObservationTitleNotificationSlackWebhook),
		ObservationTitleNotificationEmailTo:        tfutil.StringValue(m.ObservationTitleNotificationEmailTo),
		ObservationTitleNotificationMinSeverity:    tfutil.StringValue(m.ObservationTitleNotificationMinSeverity),
		ObservationTitleNotificationStatusList: emptyIfNil(
			tfutil.StringSet(ctx, m.ObservationTitleNotificationStatusList, diags)),
		ObservationTitleNotificationMinPriority: tfutil.Int64Ptr(m.ObservationTitleNotificationMinPriority),
		ObservationTitleNotificationParserType:  tfutil.StringValue(m.ObservationTitleNotificationParserType),

		BackgroundProductMetricsIntervalMinutes: m.BackgroundProductMetricsIntervalMinutes.ValueInt64(),
		BackgroundEPSSImportCrontabMinute:       m.BackgroundEPSSImportCrontabMinute.ValueInt64(),
		BackgroundEPSSImportCrontabHour:         m.BackgroundEPSSImportCrontabHour.ValueInt64(),

		BranchHousekeepingCrontabMinute:    m.BranchHousekeepingCrontabMinute.ValueInt64(),
		BranchHousekeepingCrontabHour:      m.BranchHousekeepingCrontabHour.ValueInt64(),
		BranchHousekeepingActive:           tfutil.BoolValue(m.BranchHousekeepingActive),
		BranchHousekeepingKeepInactiveDays: m.BranchHousekeepingKeepInactiveDays.ValueInt64(),
		BranchHousekeepingExemptBranches:   tfutil.StringValue(m.BranchHousekeepingExemptBranches),

		FeatureVEX:            tfutil.BoolValue(m.FeatureVEX),
		VEXJustificationStyle: tfutil.StringValue(m.VEXJustificationStyle),

		FeatureDisableUserLogin:         tfutil.BoolValue(m.FeatureDisableUserLogin),
		FeatureGeneralRulesNeedApproval: tfutil.BoolValue(m.FeatureGeneralRulesNeedApproval),

		RiskAcceptanceExpiryDays:          m.RiskAcceptanceExpiryDays.ValueInt64(),
		RiskAcceptanceExpiryCrontabMinute: m.RiskAcceptanceExpiryCrontabMinute.ValueInt64(),
		RiskAcceptanceExpiryCrontabHour:   m.RiskAcceptanceExpiryCrontabHour.ValueInt64(),

		FeatureAutomaticAPIImport: tfutil.BoolValue(m.FeatureAutomaticAPIImport),
		APIImportCrontabMinute:    m.APIImportCrontabMinute.ValueInt64(),
		APIImportCrontabHour:      m.APIImportCrontabHour.ValueInt64(),

		PasswordValidatorMinimumLength:       m.PasswordValidatorMinimumLength.ValueInt64(),
		PasswordValidatorAttributeSimilarity: tfutil.BoolValue(m.PasswordValidatorAttributeSimilarity),
		PasswordValidatorCommonPasswords:     tfutil.BoolValue(m.PasswordValidatorCommonPasswords),
		PasswordValidatorNotNumeric:          tfutil.BoolValue(m.PasswordValidatorNotNumeric),

		FeatureLicenseManagement:               tfutil.BoolValue(m.FeatureLicenseManagement),
		LicenseImportCrontabMinute:             m.LicenseImportCrontabMinute.ValueInt64(),
		LicenseImportCrontabHour:               m.LicenseImportCrontabHour.ValueInt64(),
		FeatureAutomaticOSVScanning:            tfutil.BoolValue(m.FeatureAutomaticOSVScanning),
		FeatureAutomaticVulnerableCodeScanning: tfutil.BoolValue(m.FeatureAutomaticVulnerableCodeScanning),
		VulnerableCodeBaseURL:                  tfutil.StringValue(m.VulnerableCodeBaseURL),
		VulnerableCodeAPIKey:                   tfutil.StringValue(m.VulnerableCodeAPIKey),
		FeatureExploitInformation:              tfutil.BoolValue(m.FeatureExploitInformation),
		ExploitInformationMaxAgeYears:          m.ExploitInformationMaxAgeYears.ValueInt64(),
		PeriodicTaskMaxEntries:                 m.PeriodicTaskMaxEntries.ValueInt64(),

		OIDCClockSkew:      m.OIDCClockSkew.ValueInt64(),
		OIDCStrictAudience: tfutil.BoolValue(m.OIDCStrictAudience),

		ObservationCountFromMetrics: tfutil.BoolValue(m.ObservationCountFromMetrics),

		FeatureCrossScannerDeduplication: tfutil.BoolValue(m.FeatureCrossScannerDeduplication),
		FeatureShowProductHeaderChips:    tfutil.BoolValue(m.FeatureShowProductHeaderChips),
	}
}

func (m *model) fromAPI(ctx context.Context, settings client.Settings, diags *diag.Diagnostics) {
	m.ID = types.Int64Value(1)

	m.SecurityGateActive = types.BoolValue(settings.SecurityGateActive)
	m.SecurityGateThresholdCritical = types.Int64Value(settings.SecurityGateThresholdCritical)
	m.SecurityGateThresholdHigh = types.Int64Value(settings.SecurityGateThresholdHigh)
	m.SecurityGateThresholdMedium = types.Int64Value(settings.SecurityGateThresholdMedium)
	m.SecurityGateThresholdLow = types.Int64Value(settings.SecurityGateThresholdLow)
	m.SecurityGateThresholdNone = types.Int64Value(settings.SecurityGateThresholdNone)
	m.SecurityGateThresholdUnknown = types.Int64Value(settings.SecurityGateThresholdUnknown)

	m.JWTValidityDurationUser = types.Int64Value(settings.JWTValidityDurationUser)
	m.JWTValidityDurationSuperuser = types.Int64Value(settings.JWTValidityDurationSuperuser)
	m.InternalUsers = types.StringValue(settings.InternalUsers)

	m.BaseURLFrontend = tfutil.String(settings.BaseURLFrontend)

	m.ExceptionMSTeamsWebhook = types.StringValue(settings.ExceptionMSTeamsWebhook)
	m.ExceptionSlackWebhook = types.StringValue(settings.ExceptionSlackWebhook)
	m.ExceptionRateLimit = types.Int64Value(settings.ExceptionRateLimit)
	m.EmailFrom = types.StringValue(settings.EmailFrom)
	m.ExceptionEmailTo = types.StringValue(settings.ExceptionEmailTo)

	m.ObservationTitleNotificationMSTeamsWebhook = types.StringValue(settings.ObservationTitleNotificationMSTeamsWebhook)
	m.ObservationTitleNotificationSlackWebhook = types.StringValue(settings.ObservationTitleNotificationSlackWebhook)
	m.ObservationTitleNotificationEmailTo = types.StringValue(settings.ObservationTitleNotificationEmailTo)
	m.ObservationTitleNotificationMinSeverity = types.StringValue(settings.ObservationTitleNotificationMinSeverity)
	m.ObservationTitleNotificationStatusList = tfutil.StringSetValue(ctx, settings.ObservationTitleNotificationStatusList, diags)
	m.ObservationTitleNotificationMinPriority = tfutil.Int64(settings.ObservationTitleNotificationMinPriority)
	m.ObservationTitleNotificationParserType = types.StringValue(settings.ObservationTitleNotificationParserType)

	m.BackgroundProductMetricsIntervalMinutes = types.Int64Value(settings.BackgroundProductMetricsIntervalMinutes)
	m.BackgroundEPSSImportCrontabMinute = types.Int64Value(settings.BackgroundEPSSImportCrontabMinute)
	m.BackgroundEPSSImportCrontabHour = types.Int64Value(settings.BackgroundEPSSImportCrontabHour)

	m.BranchHousekeepingCrontabMinute = types.Int64Value(settings.BranchHousekeepingCrontabMinute)
	m.BranchHousekeepingCrontabHour = types.Int64Value(settings.BranchHousekeepingCrontabHour)
	m.BranchHousekeepingActive = types.BoolValue(settings.BranchHousekeepingActive)
	m.BranchHousekeepingKeepInactiveDays = types.Int64Value(settings.BranchHousekeepingKeepInactiveDays)
	m.BranchHousekeepingExemptBranches = types.StringValue(settings.BranchHousekeepingExemptBranches)

	m.FeatureVEX = types.BoolValue(settings.FeatureVEX)
	m.VEXJustificationStyle = types.StringValue(settings.VEXJustificationStyle)

	m.FeatureDisableUserLogin = types.BoolValue(settings.FeatureDisableUserLogin)
	m.FeatureGeneralRulesNeedApproval = types.BoolValue(settings.FeatureGeneralRulesNeedApproval)

	m.RiskAcceptanceExpiryDays = types.Int64Value(settings.RiskAcceptanceExpiryDays)
	m.RiskAcceptanceExpiryCrontabMinute = types.Int64Value(settings.RiskAcceptanceExpiryCrontabMinute)
	m.RiskAcceptanceExpiryCrontabHour = types.Int64Value(settings.RiskAcceptanceExpiryCrontabHour)

	m.FeatureAutomaticAPIImport = types.BoolValue(settings.FeatureAutomaticAPIImport)
	m.APIImportCrontabMinute = types.Int64Value(settings.APIImportCrontabMinute)
	m.APIImportCrontabHour = types.Int64Value(settings.APIImportCrontabHour)

	m.PasswordValidatorMinimumLength = types.Int64Value(settings.PasswordValidatorMinimumLength)
	m.PasswordValidatorAttributeSimilarity = types.BoolValue(settings.PasswordValidatorAttributeSimilarity)
	m.PasswordValidatorCommonPasswords = types.BoolValue(settings.PasswordValidatorCommonPasswords)
	m.PasswordValidatorNotNumeric = types.BoolValue(settings.PasswordValidatorNotNumeric)

	m.FeatureLicenseManagement = types.BoolValue(settings.FeatureLicenseManagement)
	m.LicenseImportCrontabMinute = types.Int64Value(settings.LicenseImportCrontabMinute)
	m.LicenseImportCrontabHour = types.Int64Value(settings.LicenseImportCrontabHour)
	m.FeatureAutomaticOSVScanning = types.BoolValue(settings.FeatureAutomaticOSVScanning)
	m.FeatureAutomaticVulnerableCodeScanning = types.BoolValue(settings.FeatureAutomaticVulnerableCodeScanning)
	m.VulnerableCodeBaseURL = types.StringValue(settings.VulnerableCodeBaseURL)
	m.VulnerableCodeAPIKey = types.StringValue(settings.VulnerableCodeAPIKey)
	m.FeatureExploitInformation = types.BoolValue(settings.FeatureExploitInformation)
	m.ExploitInformationMaxAgeYears = types.Int64Value(settings.ExploitInformationMaxAgeYears)
	m.PeriodicTaskMaxEntries = types.Int64Value(settings.PeriodicTaskMaxEntries)

	m.OIDCClockSkew = types.Int64Value(settings.OIDCClockSkew)
	m.OIDCStrictAudience = types.BoolValue(settings.OIDCStrictAudience)

	m.ObservationCountFromMetrics = types.BoolValue(settings.ObservationCountFromMetrics)

	m.FeatureCrossScannerDeduplication = types.BoolValue(settings.FeatureCrossScannerDeduplication)
	m.FeatureShowProductHeaderChips = types.BoolValue(settings.FeatureShowProductHeaderChips)
}

func emptyIfNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
