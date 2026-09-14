package schemacommon

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tfutil"
)

// RuleFields is the writable attribute block shared by secobserve_general_rule
// and secobserve_product_rule -- everything except "product", which only
// applies to the latter, and the approval fields, which are Computed-only
// (see RuleApproval below).
type RuleFields struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Type        types.String `tfsdk:"type"`
	Parser      types.Int64  `tfsdk:"parser"`

	ScannerPrefix                     types.String `tfsdk:"scanner_prefix"`
	Title                             types.String `tfsdk:"title"`
	DescriptionObservation            types.String `tfsdk:"description_observation"`
	OriginComponentNameVersion        types.String `tfsdk:"origin_component_name_version"`
	OriginComponentPurl               types.String `tfsdk:"origin_component_purl"`
	OriginDockerImageNameTag          types.String `tfsdk:"origin_docker_image_name_tag"`
	OriginEndpointURL                 types.String `tfsdk:"origin_endpoint_url"`
	OriginServiceName                 types.String `tfsdk:"origin_service_name"`
	OriginSourceFile                  types.String `tfsdk:"origin_source_file"`
	OriginCloudQualifiedResource      types.String `tfsdk:"origin_cloud_qualified_resource"`
	OriginKubernetesQualifiedResource types.String `tfsdk:"origin_kubernetes_qualified_resource"`

	NewSeverity         types.String          `tfsdk:"new_severity"`
	NewStatus           types.String          `tfsdk:"new_status"`
	NewVEXJustification types.String          `tfsdk:"new_vex_justification"`
	NewVEXRemediations  []VEXRemediationModel `tfsdk:"new_vex_remediations"`

	RegoModule types.String `tfsdk:"rego_module"`

	Enabled types.Bool `tfsdk:"enabled"`
}

// VEXRemediationModel is one entry of new_vex_remediations.
type VEXRemediationModel struct {
	Category types.String `tfsdk:"category"`
	Text     types.String `tfsdk:"text"`
}

// AddRuleFields contributes the attributes shared by general and product
// rules.
func AddRuleFields(attributes map[string]schema.Attribute) {
	attributes["name"] = schema.StringAttribute{
		Required:            true,
		Validators:          []validator.String{stringvalidator.LengthBetween(1, 255)},
		MarkdownDescription: "Name of the rule.",
	}
	attributes["description"] = schema.StringAttribute{
		// The model declares this blank=True, but the serializer's
		// validate_description rejects an empty value outright
		// ("Must be set") -- required in practice despite the model.
		Required:            true,
		Validators:          []validator.String{stringvalidator.LengthBetween(1, 2048)},
		MarkdownDescription: "Free-text description. SecObserve requires a non-empty value here.",
	}
	attributes["type"] = schema.StringAttribute{
		Optional: true,
		Computed: true,
		Default:  stringdefault.StaticString("Fields"),
		Validators: []validator.String{
			stringvalidator.OneOf(client.RuleTypes...),
		},
		MarkdownDescription: "How the rule matches: `Fields` compares the `origin_*`/`title`/etc " +
			"attributes below against each observation, `Rego` evaluates `rego_module` instead. One of " +
			MarkdownList(client.RuleTypes) + ".",
	}
	attributes["parser"] = schema.Int64Attribute{
		Optional:   true,
		Validators: []validator.Int64{int64validator.AtLeast(1)},
		MarkdownDescription: "Only match observations from this parser. Use the `secobserve_parser` data " +
			"source to resolve a parser name to an id.",
	}

	for attribute, spec := range map[string]struct {
		maxLength int
		desc      string
	}{
		"scanner_prefix":                       {255, "Only match observations whose scanner prefix equals this."},
		"title":                                {255, "Regular expression matched against the observation's title."},
		"description_observation":              {255, "Regular expression matched against the observation's description."},
		"origin_component_name_version":        {513, "Regular expression matched against \"<component name>:<version>\"."},
		"origin_component_purl":                {255, "Regular expression matched against the component's package URL."},
		"origin_docker_image_name_tag":         {513, "Regular expression matched against \"<image name>:<tag>\"."},
		"origin_service_name":                  {255, "Regular expression matched against the origin service's name."},
		"origin_source_file":                   {255, "Regular expression matched against the origin source file path."},
		"origin_cloud_qualified_resource":      {255, "Regular expression matched against the origin cloud resource."},
		"origin_kubernetes_qualified_resource": {255, "Regular expression matched against the origin Kubernetes resource."},
	} {
		attributes[attribute] = schema.StringAttribute{
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
			Validators:          []validator.String{stringvalidator.LengthAtMost(spec.maxLength)},
			MarkdownDescription: spec.desc,
		}
	}
	// origin_endpoint_url is a TextField(max_length=2048), not the 255/513
	// CharFields above.
	attributes["origin_endpoint_url"] = schema.StringAttribute{
		Optional:            true,
		Computed:            true,
		Default:             stringdefault.StaticString(""),
		Validators:          []validator.String{stringvalidator.LengthAtMost(2048)},
		MarkdownDescription: "Regular expression matched against the origin endpoint URL.",
	}

	attributes["new_severity"] = schema.StringAttribute{
		Optional: true, Computed: true, Default: stringdefault.StaticString(""),
		Validators: []validator.String{stringvalidator.OneOf(WithEmpty(client.Severities)...)},
		MarkdownDescription: "Severity to set on a matching observation. One of " +
			MarkdownList(client.Severities) + ". The empty string leaves severity unchanged.",
	}
	attributes["new_status"] = schema.StringAttribute{
		Optional: true, Computed: true, Default: stringdefault.StaticString(""),
		Validators: []validator.String{stringvalidator.OneOf(WithEmpty(client.Statuses)...)},
		MarkdownDescription: "Status to set on a matching observation. One of " +
			MarkdownList(client.Statuses) + ". The empty string leaves status unchanged.",
	}
	attributes["new_vex_justification"] = schema.StringAttribute{
		Optional: true, Computed: true, Default: stringdefault.StaticString(""),
		Validators: []validator.String{stringvalidator.OneOf(WithEmpty(client.VEXJustifications)...)},
		MarkdownDescription: "VEX justification to set on a matching observation. One of " +
			MarkdownList(client.VEXJustifications) + ". The empty string leaves it unchanged.",
	}
	attributes["new_vex_remediations"] = schema.ListNestedAttribute{
		Optional: true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"category": schema.StringAttribute{
					Required:            true,
					MarkdownDescription: "Remediation category.",
				},
				"text": schema.StringAttribute{
					Required:            true,
					MarkdownDescription: "Remediation text.",
				},
			},
		},
		MarkdownDescription: "VEX remediations to attach to a matching observation.\n\n" +
			"~> Omit the attribute entirely to leave remediations unchanged. An empty list is **not** " +
			"valid: SecObserve stores it as null, which would leave Terraform with a permanent diff, so " +
			"the provider rejects it at plan time.",
	}

	attributes["rego_module"] = schema.StringAttribute{
		Optional: true,
		Computed: true,
		Default:  stringdefault.StaticString(""),
		MarkdownDescription: "Rego module evaluated against each observation when `type` is `Rego`. " +
			"Required when `type` is `Rego`, unused otherwise.",
	}

	attributes["enabled"] = schema.BoolAttribute{
		Optional:            true,
		Computed:            true,
		Default:             booldefault.StaticBool(true),
		MarkdownDescription: "Whether the rule is evaluated at all.",
	}
}

// ValidateRuleFields mirrors the server's cross-field rules so they fail at
// plan time: rego_module is required when type is Rego
// (rules/api/serializers.py:56-60), and an empty new_vex_remediations list
// would be silently normalized to null server-side
// (commons/services/functions.py:52-86).
func (r RuleFields) ValidateRuleFields(diags *diag.Diagnostics) {
	if r.Type.ValueString() == client.RuleTypeRego && tfutil.StringValue(r.RegoModule) == "" {
		diags.AddAttributeError(
			path.Root("rego_module"),
			"rego_module is required when type is \"Rego\"",
			"SecObserve rejects a Rego rule with no rego_module (\"Rego module must be set\").",
		)
	}

	if r.NewVEXRemediations != nil && len(r.NewVEXRemediations) == 0 {
		diags.AddAttributeError(
			path.Root("new_vex_remediations"),
			"Empty new_vex_remediations list",
			"SecObserve stores an empty remediations list as null, so Terraform would report a diff on "+
				"every plan. Omit the attribute instead of setting it to [].",
		)
	}
}

// ToAPI converts the block into its request representation.
func (r RuleFields) ToAPI() client.RuleFields {
	fields := client.RuleFields{
		Name:                              r.Name.ValueString(),
		Description:                       r.Description.ValueString(),
		Type:                              tfutil.StringValue(r.Type),
		Parser:                            tfutil.Int64Ptr(r.Parser),
		ScannerPrefix:                     tfutil.StringValue(r.ScannerPrefix),
		Title:                             tfutil.StringValue(r.Title),
		DescriptionObservation:            tfutil.StringValue(r.DescriptionObservation),
		OriginComponentNameVersion:        tfutil.StringValue(r.OriginComponentNameVersion),
		OriginComponentPurl:               tfutil.StringValue(r.OriginComponentPurl),
		OriginDockerImageNameTag:          tfutil.StringValue(r.OriginDockerImageNameTag),
		OriginEndpointURL:                 tfutil.StringValue(r.OriginEndpointURL),
		OriginServiceName:                 tfutil.StringValue(r.OriginServiceName),
		OriginSourceFile:                  tfutil.StringValue(r.OriginSourceFile),
		OriginCloudQualifiedResource:      tfutil.StringValue(r.OriginCloudQualifiedResource),
		OriginKubernetesQualifiedResource: tfutil.StringValue(r.OriginKubernetesQualifiedResource),
		NewSeverity:                       tfutil.StringValue(r.NewSeverity),
		NewStatus:                         tfutil.StringValue(r.NewStatus),
		NewVEXJustification:               tfutil.StringValue(r.NewVEXJustification),
		RegoModule:                        tfutil.StringValue(r.RegoModule),
		Enabled:                           tfutil.BoolValue(r.Enabled),
	}
	for _, remediation := range r.NewVEXRemediations {
		fields.NewVEXRemediations = append(fields.NewVEXRemediations, client.VEXRemediation{
			Category: remediation.Category.ValueString(),
			Text:     remediation.Text.ValueString(),
		})
	}
	return fields
}

// FromAPI fills the block from an API response.
func (r *RuleFields) FromAPI(fields client.RuleFields) {
	r.Name = types.StringValue(fields.Name)
	r.Description = types.StringValue(fields.Description)
	r.Type = types.StringValue(fields.Type)
	r.Parser = tfutil.Int64(fields.Parser)
	r.ScannerPrefix = types.StringValue(fields.ScannerPrefix)
	r.Title = types.StringValue(fields.Title)
	r.DescriptionObservation = types.StringValue(fields.DescriptionObservation)
	r.OriginComponentNameVersion = types.StringValue(fields.OriginComponentNameVersion)
	r.OriginComponentPurl = types.StringValue(fields.OriginComponentPurl)
	r.OriginDockerImageNameTag = types.StringValue(fields.OriginDockerImageNameTag)
	r.OriginEndpointURL = types.StringValue(fields.OriginEndpointURL)
	r.OriginServiceName = types.StringValue(fields.OriginServiceName)
	r.OriginSourceFile = types.StringValue(fields.OriginSourceFile)
	r.OriginCloudQualifiedResource = types.StringValue(fields.OriginCloudQualifiedResource)
	r.OriginKubernetesQualifiedResource = types.StringValue(fields.OriginKubernetesQualifiedResource)
	r.NewSeverity = types.StringValue(fields.NewSeverity)
	r.NewStatus = types.StringValue(fields.NewStatus)
	r.NewVEXJustification = types.StringValue(fields.NewVEXJustification)
	r.RegoModule = types.StringValue(fields.RegoModule)
	r.Enabled = types.BoolValue(fields.Enabled)

	// Null and [] are the same thing here, and null is what SecObserve stores.
	if len(fields.NewVEXRemediations) == 0 {
		r.NewVEXRemediations = nil
	} else {
		r.NewVEXRemediations = make([]VEXRemediationModel, 0, len(fields.NewVEXRemediations))
		for _, remediation := range fields.NewVEXRemediations {
			r.NewVEXRemediations = append(r.NewVEXRemediations, VEXRemediationModel{
				Category: types.StringValue(remediation.Category),
				Text:     types.StringValue(remediation.Text),
			})
		}
	}
}

// RuleApproval is the Computed-only approval workflow block, identical for
// general and product rules.
//
// Every one of these fields is server-owned: any create or update resets
// ApprovalStatus and reassigns User (rules/models.py Rule.save():75-101).
// Approving a rule is a separate, out-of-band action -- see the resource
// documentation -- and the identity that created a rule can never approve it
// (rules/services/approval.py:16-17 rejects self-approval).
type RuleApproval struct {
	User                 types.String `tfsdk:"user"`
	UserFullName         types.String `tfsdk:"user_full_name"`
	ApprovalStatus       types.String `tfsdk:"approval_status"`
	RejectionRemark      types.String `tfsdk:"rejection_remark"`
	ApprovalDate         types.String `tfsdk:"approval_date"`
	ApprovalUser         types.String `tfsdk:"approval_user"`
	ApprovalUserFullName types.String `tfsdk:"approval_user_full_name"`
}

// AddRuleApproval contributes the approval workflow attributes.
func AddRuleApproval(attributes map[string]schema.Attribute) {
	attributes["user"] = schema.StringAttribute{
		Computed:            true,
		MarkdownDescription: "Username of whoever created or last modified this rule.",
	}
	attributes["user_full_name"] = schema.StringAttribute{
		Computed:            true,
		MarkdownDescription: "Display name of `user`.",
	}
	attributes["approval_status"] = schema.StringAttribute{
		Computed: true,
		MarkdownDescription: "`Needs approval`, `Approved`, `Auto approved` or `Rejected`.\n\n" +
			"~> Any create or update resets this to `Needs approval` or `Auto approved` depending on the " +
			"instance and product approval settings, and reassigns `user` to whoever ran `terraform " +
			"apply`. Approval itself happens outside Terraform: `PATCH .../approval/` is a separate call, " +
			"and SecObserve rejects self-approval, so the identity that applied this resource can never " +
			"approve the rule it just wrote.",
	}
	attributes["rejection_remark"] = schema.StringAttribute{
		Computed:            true,
		MarkdownDescription: "Reason given when this rule was last rejected, if any.",
	}
	attributes["approval_date"] = schema.StringAttribute{
		Computed:            true,
		MarkdownDescription: "When this rule was last approved or rejected.",
	}
	attributes["approval_user"] = schema.StringAttribute{
		Computed:            true,
		MarkdownDescription: "Username of whoever last approved or rejected this rule.",
	}
	attributes["approval_user_full_name"] = schema.StringAttribute{
		Computed:            true,
		MarkdownDescription: "Display name of `approval_user`.",
	}
}

// FromAPI fills the block from an API response.
func (a *RuleApproval) FromAPI(rule client.Rule) {
	a.User = types.StringValue(rule.User)
	a.UserFullName = tfutil.String(rule.UserFullName)
	a.ApprovalStatus = types.StringValue(rule.ApprovalStatus)
	a.RejectionRemark = types.StringValue(rule.RejectionRemark)
	a.ApprovalDate = tfutil.String(rule.ApprovalDate)
	a.ApprovalUser = tfutil.String(rule.ApprovalUser)
	a.ApprovalUserFullName = tfutil.String(rule.ApprovalUserFullName)
}
