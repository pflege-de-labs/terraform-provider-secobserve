package settings

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/schemacommon"
	sovalidators "github.com/pflege-de-labs/terraform-provider-secobserve/internal/validators"
)

// crontabMinute and crontabHour build the paired attributes for the several
// "run this job at HH:MM UTC" settings, which all share the same shape.
func crontabMinute(defaultValue int64, job string) schema.Attribute {
	return schema.Int64Attribute{
		Optional:            true,
		Computed:            true,
		Default:             int64default.StaticInt64(defaultValue),
		Validators:          []validator.Int64{int64validator.Between(0, 59)},
		MarkdownDescription: "Minute (UTC) " + job + " runs at.",
	}
}

func crontabHour(defaultValue int64, job string) schema.Attribute {
	return schema.Int64Attribute{
		Optional:            true,
		Computed:            true,
		Default:             int64default.StaticInt64(defaultValue),
		Validators:          []validator.Int64{int64validator.Between(0, 23)},
		MarkdownDescription: "Hour (UTC) " + job + " runs at.",
	}
}

func stringSetting(defaultValue string) schema.Attribute {
	return schema.StringAttribute{
		Optional: true,
		Computed: true,
		Default:  stringdefault.StaticString(defaultValue),
	}
}

func boolSetting(defaultValue bool) schema.Attribute {
	return schema.BoolAttribute{
		Optional: true,
		Computed: true,
		Default:  booldefault.StaticBool(defaultValue),
	}
}

func (r *settingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := map[string]schema.Attribute{
		"id": schema.Int64Attribute{
			Computed: true,
			PlanModifiers: []planmodifier.Int64{
				schemacommon.Int64UseStateForUnknown(),
			},
			MarkdownDescription: "Always `1`: SecObserve keeps exactly one settings record and the API " +
				"hardcodes its id.",
		},

		// Security gate defaults, inherited by any product or product group
		// that leaves its own security_gate_* attributes unset.
		"security_gate_active": schema.BoolAttribute{
			Optional: true, Computed: true, Default: booldefault.StaticBool(true),
			MarkdownDescription: "Instance-wide default for whether the security gate is evaluated.",
		},
		"security_gate_threshold_critical": schema.Int64Attribute{
			Optional: true, Computed: true, Default: int64default.StaticInt64(0),
			Validators:          []validator.Int64{int64validator.Between(0, 999999)},
			MarkdownDescription: "Instance-wide default critical-severity threshold.",
		},
		"security_gate_threshold_high": schema.Int64Attribute{
			Optional: true, Computed: true, Default: int64default.StaticInt64(0),
			Validators:          []validator.Int64{int64validator.Between(0, 999999)},
			MarkdownDescription: "Instance-wide default high-severity threshold.",
		},
		"security_gate_threshold_medium": schema.Int64Attribute{
			Optional: true, Computed: true, Default: int64default.StaticInt64(99999),
			Validators:          []validator.Int64{int64validator.Between(0, 999999)},
			MarkdownDescription: "Instance-wide default medium-severity threshold.",
		},
		"security_gate_threshold_low": schema.Int64Attribute{
			Optional: true, Computed: true, Default: int64default.StaticInt64(99999),
			Validators:          []validator.Int64{int64validator.Between(0, 999999)},
			MarkdownDescription: "Instance-wide default low-severity threshold.",
		},
		"security_gate_threshold_none": schema.Int64Attribute{
			Optional: true, Computed: true, Default: int64default.StaticInt64(99999),
			Validators:          []validator.Int64{int64validator.Between(0, 999999)},
			MarkdownDescription: "Instance-wide default `None`-severity threshold.",
		},
		"security_gate_threshold_unknown": schema.Int64Attribute{
			Optional: true, Computed: true, Default: int64default.StaticInt64(99999),
			Validators:          []validator.Int64{int64validator.Between(0, 999999)},
			MarkdownDescription: "Instance-wide default unknown-severity threshold.",
		},

		"jwt_validity_duration_user": schema.Int64Attribute{
			Optional: true, Computed: true, Default: int64default.StaticInt64(168),
			Validators:          []validator.Int64{int64validator.Between(0, 999999)},
			MarkdownDescription: "Hours a regular user's frontend JWT stays valid.",
		},
		"jwt_validity_duration_superuser": schema.Int64Attribute{
			Optional: true, Computed: true, Default: int64default.StaticInt64(24),
			Validators:          []validator.Int64{int64validator.Between(0, 999999)},
			MarkdownDescription: "Hours a superuser's frontend JWT stays valid.",
		},
		"internal_users": schema.StringAttribute{
			Optional: true, Computed: true, Default: stringdefault.StaticString(""),
			Validators: []validator.String{stringvalidator.LengthAtMost(255), sovalidators.RegularExpression()},
			MarkdownDescription: "Comma-separated regular expressions matched against an OIDC user's email " +
				"to decide whether they are internal. Evaluated only when a user is first created; changing " +
				"it does not reclassify existing users.",
		},

		"base_url_frontend": schema.StringAttribute{
			// No static default, unlike every other string setting here:
			// unlike them, this one has no blank=True on the model
			// (commons/models.py:69-73) and the API rejects an explicit ""
			// outright, so a default that could resolve to "" would break
			// every apply that leaves this unset. Leaving it Computed with
			// no default means an unset attribute is simply never sent, and
			// SecObserve leaves the current value alone -- see
			// client.Settings.BaseURLFrontend.
			Optional:   true,
			Computed:   true,
			Validators: []validator.String{stringvalidator.LengthBetween(1, 255), sovalidators.HTTPURL()},
			MarkdownDescription: "Base URL of the frontend, used to build links in notifications.\n\n" +
				"~> Unlike every other URL setting here, SecObserve rejects an explicit empty string for " +
				"this one (`This field may not be blank`), so it cannot be cleared back to \"\" once set.",
		},

		"exception_ms_teams_webhook": schema.StringAttribute{
			Optional: true, Computed: true, Default: stringdefault.StaticString(""),
			Validators:          []validator.String{stringvalidator.LengthAtMost(2048), sovalidators.HTTPURL()},
			MarkdownDescription: "Microsoft Teams webhook for exception notifications.",
		},
		"exception_slack_webhook": schema.StringAttribute{
			Optional: true, Computed: true, Default: stringdefault.StaticString(""),
			Validators:          []validator.String{stringvalidator.LengthAtMost(2048), sovalidators.HTTPURL()},
			MarkdownDescription: "Slack webhook for exception notifications.",
		},
		"exception_rate_limit": schema.Int64Attribute{
			Optional: true, Computed: true, Default: int64default.StaticInt64(3600),
			Validators:          []validator.Int64{int64validator.Between(0, 999999)},
			MarkdownDescription: "Seconds before the same exception can trigger another notification.",
		},
		"email_from": stringSetting(""),
		"exception_email_to": schema.StringAttribute{
			Optional: true, Computed: true, Default: stringdefault.StaticString(""),
			Validators:          []validator.String{stringvalidator.LengthAtMost(255)},
			MarkdownDescription: "Comma-separated email addresses for exception notifications.",
		},

		"observation_title_notification_ms_teams_webhook": schema.StringAttribute{
			Optional: true, Computed: true, Default: stringdefault.StaticString(""),
			Validators:          []validator.String{stringvalidator.LengthAtMost(2048), sovalidators.HTTPURL()},
			MarkdownDescription: "Microsoft Teams webhook for observation-title notifications.",
		},
		"observation_title_notification_slack_webhook": schema.StringAttribute{
			Optional: true, Computed: true, Default: stringdefault.StaticString(""),
			Validators:          []validator.String{stringvalidator.LengthAtMost(2048), sovalidators.HTTPURL()},
			MarkdownDescription: "Slack webhook for observation-title notifications.",
		},
		"observation_title_notification_email_to": schema.StringAttribute{
			Optional: true, Computed: true, Default: stringdefault.StaticString(""),
			Validators:          []validator.String{stringvalidator.LengthAtMost(255)},
			MarkdownDescription: "Comma-separated email addresses for observation-title notifications.",
		},
		"observation_title_notification_min_severity": schema.StringAttribute{
			Optional: true, Computed: true, Default: stringdefault.StaticString(""),
			Validators: []validator.String{stringvalidator.OneOf(schemacommon.WithEmpty(client.Severities)...)},
			MarkdownDescription: "Only notify about a new observation title of at least this severity. One " +
				"of " + schemacommon.MarkdownList(client.Severities) + ". The empty string disables the " +
				"severity condition.",
		},
		"observation_title_notification_status_list": schema.SetAttribute{
			Optional: true, Computed: true, ElementType: types.StringType,
			Default: setdefault.StaticValue(schemacommon.EmptyStringSet()),
			Validators: []validator.Set{
				setvalidator.ValueStringsAre(stringvalidator.OneOf(client.Statuses...)),
			},
			MarkdownDescription: "Only notify about a new observation title in one of these statuses. Any " +
				"of " + schemacommon.MarkdownList(client.Statuses) + ". An empty set disables the status " +
				"condition.",
		},
		"observation_title_notification_min_priority": schema.Int64Attribute{
			Optional:            true,
			Validators:          []validator.Int64{int64validator.Between(1, 99)},
			MarkdownDescription: "Only notify about a new observation title with at least this priority (1-99).",
		},
		"observation_title_notification_parser_type": schema.StringAttribute{
			Optional: true, Computed: true, Default: stringdefault.StaticString(""),
			Validators: []validator.String{stringvalidator.OneOf(schemacommon.WithEmpty(client.ParserTypes)...)},
			MarkdownDescription: "Only notify about a new observation title from this scanner category. One " +
				"of " + schemacommon.MarkdownList(client.ParserTypes) + ".",
		},

		"background_product_metrics_interval_minutes": schema.Int64Attribute{
			Optional: true, Computed: true, Default: int64default.StaticInt64(5),
			Validators:          []validator.Int64{int64validator.Between(0, 999999)},
			MarkdownDescription: "How often product metrics are recalculated, in minutes.",
		},
		"background_epss_import_crontab_minute": crontabMinute(0, "the EPSS/cvss-bt import"),
		"background_epss_import_crontab_hour":   crontabHour(3, "the EPSS/cvss-bt import"),

		"branch_housekeeping_crontab_minute": crontabMinute(0, "branch housekeeping"),
		"branch_housekeeping_crontab_hour":   crontabHour(2, "branch housekeeping"),
		"branch_housekeeping_active":         boolSetting(true),
		"branch_housekeeping_keep_inactive_days": schema.Int64Attribute{
			Optional: true, Computed: true, Default: int64default.StaticInt64(30),
			Validators:          []validator.Int64{int64validator.Between(0, 999999)},
			MarkdownDescription: "Instance-wide default days a branch may stay inactive before deletion.",
		},
		"branch_housekeeping_exempt_branches": schema.StringAttribute{
			Optional: true, Computed: true, Default: stringdefault.StaticString(""),
			Validators:          []validator.String{stringvalidator.LengthAtMost(255), sovalidators.RegularExpression()},
			MarkdownDescription: "Instance-wide default regular expression exempting branches from housekeeping.",
		},

		"feature_vex": boolSetting(false),
		"vex_justification_style": schema.StringAttribute{
			Optional: true, Computed: true, Default: stringdefault.StaticString("CSAF/OpenVEX"),
			Validators: []validator.String{stringvalidator.OneOf(client.VEXJustificationStyles...)},
			MarkdownDescription: "VEX justification vocabulary to use: one of " +
				schemacommon.MarkdownList(client.VEXJustificationStyles) + ".",
		},

		"feature_disable_user_login": boolSetting(false),
		"feature_general_rules_need_approval": schema.BoolAttribute{
			Optional: true, Computed: true, Default: booldefault.StaticBool(false),
			MarkdownDescription: "Require approval for general rules. A rule created while this is enabled " +
				"stays inactive until somebody other than its author approves it, and SecObserve rejects " +
				"self-approval -- the provider can never approve its own rules.",
		},

		"risk_acceptance_expiry_days": schema.Int64Attribute{
			Optional: true, Computed: true, Default: int64default.StaticInt64(30),
			Validators:          []validator.Int64{int64validator.Between(0, 999999)},
			MarkdownDescription: "Instance-wide default days before an accepted risk expires. `0` means never.",
		},
		"risk_acceptance_expiry_crontab_minute": crontabMinute(0, "the risk acceptance expiry check"),
		"risk_acceptance_expiry_crontab_hour":   crontabHour(1, "the risk acceptance expiry check"),

		"feature_automatic_api_import": boolSetting(true),
		"api_import_crontab_minute":    crontabMinute(0, "automatic API imports"),
		"api_import_crontab_hour":      crontabHour(4, "automatic API imports"),

		"password_validator_minimum_length": schema.Int64Attribute{
			Optional: true, Computed: true, Default: int64default.StaticInt64(8),
			Validators:          []validator.Int64{int64validator.Between(1, 4096)},
			MarkdownDescription: "Minimum password length.",
		},
		"password_validator_attribute_similarity": boolSetting(true),
		"password_validator_common_passwords":     boolSetting(true),
		"password_validator_not_numeric":          boolSetting(true),

		"feature_license_management": schema.BoolAttribute{
			Optional: true, Computed: true, Default: booldefault.StaticBool(true),
			MarkdownDescription: "Enable license management.\n\n" +
				"~> SecObserve force-enables this whenever `feature_automatic_osv_scanning` is `true` " +
				"(the default), regardless of what is sent here in the same request. Since Terraform " +
				"forbids a provider from applying a different value than the one planned, `false` here is " +
				"**rejected at plan time** rather than accepted and silently overridden -- set " +
				"`feature_automatic_osv_scanning = false` too, or drop this attribute.",
		},
		"license_import_crontab_minute":             crontabMinute(30, "the license import"),
		"license_import_crontab_hour":               crontabHour(1, "the license import"),
		"feature_automatic_osv_scanning":            boolSetting(true),
		"feature_automatic_vulnerablecode_scanning": boolSetting(false),
		"vulnerablecode_base_url": schema.StringAttribute{
			Optional: true, Computed: true, Default: stringdefault.StaticString(""),
			Validators:          []validator.String{stringvalidator.LengthAtMost(255), sovalidators.HTTPURL()},
			MarkdownDescription: "Base URL of the VulnerableCode instance.",
		},
		"vulnerablecode_api_key": schema.StringAttribute{
			Optional: true, Computed: true, Sensitive: true, Default: stringdefault.StaticString(""),
			Validators: []validator.String{stringvalidator.LengthAtMost(255)},
			MarkdownDescription: "API key for the VulnerableCode instance. Encrypted at rest; SecObserve " +
				"returns it in plaintext to a superuser caller.",
		},
		"feature_exploit_information": boolSetting(true),
		"exploit_information_max_age_years": schema.Int64Attribute{
			Optional: true, Computed: true, Default: int64default.StaticInt64(10),
			Validators:          []validator.Int64{int64validator.Between(0, 999999)},
			MarkdownDescription: "Maximum CVE age considered for exploit-information enrichment, in years.",
		},
		"periodic_task_max_entries": schema.Int64Attribute{
			Optional: true, Computed: true, Default: int64default.StaticInt64(10),
			Validators:          []validator.Int64{int64validator.Between(1, 999999)},
			MarkdownDescription: "How many run records to keep per periodic task.",
		},

		"oidc_clock_skew": schema.Int64Attribute{
			Optional: true, Computed: true, Default: int64default.StaticInt64(0),
			Validators:          []validator.Int64{int64validator.Between(0, 999999)},
			MarkdownDescription: "Seconds of clock skew tolerated when validating OIDC token timestamps.",
		},

		"oidc_strict_audience": schema.BoolAttribute{
			Optional: true, Computed: true, Default: booldefault.StaticBool(true),
			MarkdownDescription: "Require an OIDC token's `aud` claim to be a single string matching the " +
				"client id. Disable only for identity providers that issue multi-valued audiences.",
		},

		"observation_count_from_metrics": boolSetting(false),

		"feature_cross_scanner_deduplication": boolSetting(false),
		"feature_show_product_header_chips":   boolSetting(false),
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the SecObserve instance settings singleton (`GET`/`PATCH " +
			"/api/settings/1/`), superuser-only.\n\n" +
			"There is exactly one settings record, created implicitly by SecObserve and never deleted: " +
			"`terraform destroy` on this resource only removes it from Terraform state, it does not reset " +
			"anything on the server. Every attribute is `Optional` with the server's own default, so " +
			"omitting an attribute here does not clobber a value set outside Terraform -- but this also " +
			"means a fresh `terraform apply` with everything omitted still pins every setting to that " +
			"default from then on.\n\n" +
			"Import with any id, for example `terraform import secobserve_settings.this 1`.",
		Attributes: attributes,
	}
}
