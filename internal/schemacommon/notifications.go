package schemacommon

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tfutil"
	sovalidators "github.com/pflege-de-labs/terraform-provider-secobserve/internal/validators"
)

// Notifications is the observation notification block of a product or product
// group.
type Notifications struct {
	NotificationMSTeamsWebhook types.String `tfsdk:"notification_ms_teams_webhook"`
	NotificationSlackWebhook   types.String `tfsdk:"notification_slack_webhook"`
	NotificationEmailTo        types.String `tfsdk:"notification_email_to"`

	ObservationNotificationMinSeverity types.String `tfsdk:"observation_notification_min_severity"`
	ObservationNotificationStatusList  types.Set    `tfsdk:"observation_notification_status_list"`
	ObservationNotificationMinPriority types.Int64  `tfsdk:"observation_notification_min_priority"`
}

// AddNotifications contributes the notification attributes.
func AddNotifications(attributes map[string]schema.Attribute) {
	attributes["notification_ms_teams_webhook"] = schema.StringAttribute{
		Optional: true,
		Computed: true,
		Default:  stringdefault.StaticString(""),
		Validators: []validator.String{
			stringvalidator.LengthAtMost(2048),
			sovalidators.HTTPURL(),
		},
		MarkdownDescription: "Microsoft Teams incoming webhook URL for observation notifications.",
	}
	attributes["notification_slack_webhook"] = schema.StringAttribute{
		Optional: true,
		Computed: true,
		Default:  stringdefault.StaticString(""),
		Validators: []validator.String{
			stringvalidator.LengthAtMost(2048),
			sovalidators.HTTPURL(),
		},
		MarkdownDescription: "Slack incoming webhook URL for observation notifications.",
	}
	attributes["notification_email_to"] = schema.StringAttribute{
		Optional:            true,
		Computed:            true,
		Default:             stringdefault.StaticString(""),
		Validators:          []validator.String{stringvalidator.LengthAtMost(255)},
		MarkdownDescription: "Comma-separated list of email addresses for observation notifications.",
	}

	attributes["observation_notification_min_severity"] = schema.StringAttribute{
		Optional:   true,
		Computed:   true,
		Default:    stringdefault.StaticString(""),
		Validators: []validator.String{stringvalidator.OneOf(WithEmpty(client.Severities)...)},
		MarkdownDescription: "Only notify about observations of at least this severity. One of " +
			MarkdownList(client.Severities) + ". The empty string disables the severity condition.\n\n" +
			"Note that `None` is a severity level in SecObserve, not the absence of one.",
	}
	attributes["observation_notification_status_list"] = schema.SetAttribute{
		Optional:    true,
		Computed:    true,
		ElementType: types.StringType,
		Default:     setdefault.StaticValue(EmptyStringSet()),
		Validators: []validator.Set{
			setvalidator.ValueStringsAre(stringvalidator.OneOf(client.Statuses...)),
		},
		MarkdownDescription: "Only notify about observations in one of these statuses. Any of " +
			MarkdownList(client.Statuses) + ". An empty set disables the status condition.",
	}
	attributes["observation_notification_min_priority"] = schema.Int64Attribute{
		Optional:            true,
		Validators:          []validator.Int64{int64validator.Between(1, 99)},
		MarkdownDescription: "Only notify about observations with at least this priority (1-99).",
	}
}

// ToAPI converts the block into its request representation.
func (n Notifications) ToAPI(ctx context.Context, diags *diag.Diagnostics) client.NotificationFields {
	return client.NotificationFields{
		NotificationMSTeamsWebhook:         tfutil.StringValue(n.NotificationMSTeamsWebhook),
		NotificationSlackWebhook:           tfutil.StringValue(n.NotificationSlackWebhook),
		NotificationEmailTo:                tfutil.StringValue(n.NotificationEmailTo),
		ObservationNotificationMinSeverity: tfutil.StringValue(n.ObservationNotificationMinSeverity),
		ObservationNotificationStatusList:  emptyIfNil(tfutil.StringSet(ctx, n.ObservationNotificationStatusList, diags)),
		ObservationNotificationMinPriority: tfutil.Int64Ptr(n.ObservationNotificationMinPriority),
	}
}

// FromAPI fills the block from an API response.
func (n *Notifications) FromAPI(ctx context.Context, fields client.NotificationFields, diags *diag.Diagnostics) {
	n.NotificationMSTeamsWebhook = types.StringValue(fields.NotificationMSTeamsWebhook)
	n.NotificationSlackWebhook = types.StringValue(fields.NotificationSlackWebhook)
	n.NotificationEmailTo = types.StringValue(fields.NotificationEmailTo)
	n.ObservationNotificationMinSeverity = types.StringValue(fields.ObservationNotificationMinSeverity)
	n.ObservationNotificationStatusList = tfutil.StringSetValue(ctx, fields.ObservationNotificationStatusList, diags)
	n.ObservationNotificationMinPriority = tfutil.Int64(fields.ObservationNotificationMinPriority)
}

// WithEmpty allows clearing an enum attribute, which SecObserve models as the
// empty string rather than null.
func WithEmpty(values []string) []string {
	return append([]string{""}, values...)
}

// emptyIfNil keeps a cleared collection as [] rather than null, which is what
// SecObserve returns and therefore what avoids a diff.
func emptyIfNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

// MarkdownList renders enum values as inline code for attribute descriptions,
// so the docs can never disagree with the validator.
func MarkdownList(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, "`"+value+"`")
	}
	return strings.Join(quoted, ", ")
}
