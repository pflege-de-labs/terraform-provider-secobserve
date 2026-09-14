// Package license_policy_item implements the secobserve_license_policy_item resource.
package license_policy_item //nolint:revive // the package name mirrors the Terraform resource name

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/schemacommon"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tfutil"
)

var (
	_ resource.ResourceWithConfigure      = (*itemResource)(nil)
	_ resource.ResourceWithImportState    = (*itemResource)(nil)
	_ resource.ResourceWithValidateConfig = (*itemResource)(nil)
)

// New returns the secobserve_license_policy_item resource.
func New() resource.Resource {
	return &itemResource{}
}

type itemResource struct {
	client *client.Client
}

type model struct {
	ID                types.Int64  `tfsdk:"id"`
	LicensePolicy     types.Int64  `tfsdk:"license_policy"`
	LicenseGroup      types.Int64  `tfsdk:"license_group"`
	License           types.Int64  `tfsdk:"license"`
	LicenseExpression types.String `tfsdk:"license_expression"`
	NonSPDXLicense    types.String `tfsdk:"non_spdx_license"`
	EvaluationResult  types.String `tfsdk:"evaluation_result"`
	Comment           types.String `tfsdk:"comment"`

	LicenseSPDXID    types.String `tfsdk:"license_spdx_id"`
	LicenseGroupName types.String `tfsdk:"license_group_name"`
}

func (r *itemResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_license_policy_item"
}

func (r *itemResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = tfutil.ResourceClient(req, resp)
}

func (r *itemResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "One allow/forbid/review rule within a `secobserve_license_policy`, matching " +
			"components by exactly one of `license_group`, `license`, `license_expression` or " +
			"`non_spdx_license`.\n\n" +
			"~> **Exactly one of `license_group`, `license`, `license_expression` and `non_spdx_license` must " +
			"be set.** SecObserve rejects zero or more than one at apply time; the provider checks this at " +
			"plan time too.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{schemacommon.Int64UseStateForUnknown()},
				MarkdownDescription: "Numeric id of the item.",
			},
			"license_policy": schema.Int64Attribute{
				Required:      true,
				Validators:    []validator.Int64{int64validator.AtLeast(1)},
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
				MarkdownDescription: "Id of the license policy this item belongs to. Immutable: a change " +
					"replaces the item.",
			},
			"license_group": schema.Int64Attribute{
				Optional:            true,
				Validators:          []validator.Int64{int64validator.AtLeast(1)},
				MarkdownDescription: "Id of the license group this item matches. Exactly one of the four match fields.",
			},
			"license": schema.Int64Attribute{
				Optional:   true,
				Validators: []validator.Int64{int64validator.AtLeast(1)},
				MarkdownDescription: "Id of the single license this item matches. Use the `secobserve_license` " +
					"data source to resolve an SPDX id. Exactly one of the four match fields.",
			},
			"license_expression": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
				MarkdownDescription: "SPDX license expression this item matches, for example `MIT OR " +
					"Apache-2.0`.\n\n" +
					"~> **Must already be in canonical SPDX form.** SecObserve accepts and silently " +
					"normalizes a differently-cased equivalent (operators upper-cased, identifiers " +
					"canonicalized) rather than rejecting it, but Terraform's own plan/apply consistency " +
					"check does not allow the applied value to differ from what was planned: submitting " +
					"anything other than the canonical form -- `mit or apache-2.0` instead of `MIT OR " +
					"Apache-2.0` -- fails the apply with \"provider produced inconsistent result after " +
					"apply\", even though the request itself succeeded. The provider does not attempt to " +
					"normalize the expression client-side to work around this.\n\n" +
					"Exactly one of the four match fields.",
			},
			"non_spdx_license": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				Validators:          []validator.String{stringvalidator.LengthAtMost(255)},
				MarkdownDescription: "Free-text license name for a license with no SPDX identifier. Exactly one of the four match fields.",
			},
			"evaluation_result": schema.StringAttribute{
				Required:   true,
				Validators: []validator.String{stringvalidator.OneOf(client.LicensePolicyEvaluationResults...)},
				MarkdownDescription: "Outcome for a matching component. One of " +
					schemacommon.MarkdownList(client.LicensePolicyEvaluationResults) + ".",
			},
			"comment": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				Validators:          []validator.String{stringvalidator.LengthAtMost(255)},
				MarkdownDescription: "Free-text comment.",
			},
			"license_spdx_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "SPDX id of `license`, or empty if this item does not match on a single license.",
			},
			"license_group_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Name of `license_group`, or empty if this item does not match on a license group.",
			},
		},
	}
}

// ValidateConfig mirrors LicensePolicyItemSerializer.validate()
// (licenses/api/serializers.py:523-534): exactly one of the four match
// fields must be set.
func (r *itemResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config model
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Any of the four fields can be a reference to another resource or data
	// source (for example license = data.secobserve_license.mit.id) that is
	// not resolved yet when ValidateConfig runs. Skip the check rather than
	// misreporting "missing" for a field that is merely unknown for now; the
	// server still enforces this rule at apply time as a backstop.
	if config.LicenseGroup.IsUnknown() || config.License.IsUnknown() ||
		config.LicenseExpression.IsUnknown() || config.NonSPDXLicense.IsUnknown() {
		return
	}

	set := 0
	if !config.LicenseGroup.IsNull() {
		set++
	}
	if !config.License.IsNull() {
		set++
	}
	if tfutil.StringValue(config.LicenseExpression) != "" {
		set++
	}
	if tfutil.StringValue(config.NonSPDXLicense) != "" {
		set++
	}

	switch {
	case set == 0:
		resp.Diagnostics.AddError(
			"Missing license policy item match field",
			"One of license_group, license, license_expression or non_spdx_license must be set.",
		)
	case set > 1:
		resp.Diagnostics.AddError(
			"Multiple license policy item match fields",
			"Only one of license_group, license, license_expression or non_spdx_license may be set.",
		)
	}
}

func (m *model) fromAPI(item client.LicensePolicyItem) {
	m.ID = types.Int64Value(item.ID)
	m.LicensePolicy = types.Int64Value(item.LicensePolicy)
	m.LicenseGroup = tfutil.Int64(item.LicenseGroup)
	m.License = tfutil.Int64(item.License)
	m.LicenseExpression = types.StringValue(item.LicenseExpression)
	m.NonSPDXLicense = types.StringValue(item.NonSPDXLicense)
	m.EvaluationResult = types.StringValue(item.EvaluationResult)
	m.Comment = types.StringValue(item.Comment)
	m.LicenseSPDXID = types.StringValue(item.LicenseSPDXID)
	m.LicenseGroupName = types.StringValue(item.LicenseGroupName)
}

// toRequest always sends all four match fields, even the unused ones as
// null/"": the server's validate() copies whatever the request carries onto
// the instance before checking "exactly one set", so a partial update would
// silently clear the other three (licenses/api/serializers.py:530-534).
func (m model) toRequest() client.LicensePolicyItemRequest {
	return client.LicensePolicyItemRequest{
		LicensePolicy:     m.LicensePolicy.ValueInt64(),
		LicenseGroup:      tfutil.Int64Ptr(m.LicenseGroup),
		License:           tfutil.Int64Ptr(m.License),
		LicenseExpression: tfutil.StringValue(m.LicenseExpression),
		NonSPDXLicense:    tfutil.StringValue(m.NonSPDXLicense),
		EvaluationResult:  m.EvaluationResult.ValueString(),
		Comment:           tfutil.StringValue(m.Comment),
	}
}

func (r *itemResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateLicensePolicyItem(ctx, plan.toRequest())
	if err != nil {
		resp.Diagnostics.AddError("Could not create SecObserve license policy item", err.Error())
		return
	}

	var state model
	state.fromAPI(created)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *itemResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	item, err := r.client.LicensePolicyItem(ctx, state.ID.ValueInt64())
	if err != nil {
		if client.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Could not read SecObserve license policy item", err.Error())
		return
	}

	var refreshed model
	refreshed.fromAPI(item)
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *itemResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateLicensePolicyItem(ctx, state.ID.ValueInt64(), plan.toRequest())
	if err != nil {
		resp.Diagnostics.AddError("Could not update SecObserve license policy item", err.Error())
		return
	}

	var refreshed model
	refreshed.fromAPI(updated)
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *itemResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteLicensePolicyItem(ctx, state.ID.ValueInt64()); err != nil && !client.NotFound(err) {
		resp.Diagnostics.AddError("Could not delete SecObserve license policy item", err.Error())
	}
}

// ImportState takes the numeric id: an item has no unique name to import by.
func (r *itemResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import id", "Import a license policy item by its numeric id.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}
