// Package authorization_group implements the secobserve_authorization_group
// resource.
package authorization_group //nolint:revive // the package name mirrors the Terraform resource name

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/schemacommon"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tfutil"
)

var (
	_ resource.ResourceWithConfigure   = (*authorizationGroupResource)(nil)
	_ resource.ResourceWithImportState = (*authorizationGroupResource)(nil)
)

// New returns the secobserve_authorization_group resource.
func New() resource.Resource {
	return &authorizationGroupResource{}
}

type authorizationGroupResource struct {
	client *client.Client
}

type model struct {
	ID        types.Int64  `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	OIDCGroup types.String `tfsdk:"oidc_group"`
}

func (r *authorizationGroupResource) Metadata(
	_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_authorization_group"
}

func (r *authorizationGroupResource) Configure(
	_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse,
) {
	r.client = tfutil.ResourceClient(req, resp)
}

func (r *authorizationGroupResource) Schema(
	_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A named group of users that can be granted roles on products and license " +
			"objects together, instead of one membership per user.\n\n" +
			"The informational flags SecObserve reports alongside a group -- whether it has members, " +
			"whether the caller manages it -- are runtime state, not configuration; read them from the " +
			"`secobserve_authorization_group` data source instead.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{schemacommon.Int64UseStateForUnknown()},
				MarkdownDescription: "Numeric id of the authorization group.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthBetween(1, 255)},
				MarkdownDescription: "Name of the authorization group. Globally unique.",
			},
			"oidc_group": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
				Validators: []validator.String{
					stringvalidator.LengthAtMost(255),
				},
				MarkdownDescription: "Maps this group to an IdP group claim: on OIDC login, SecObserve makes " +
					"the group's members exactly the set of users whose token carries this claim.\n\n" +
					"~> Setting this makes membership entirely IdP-managed. Any " +
					"`secobserve_authorization_group_member` referencing this group will have its " +
					"membership **deleted at the member's next login** by SecObserve's group sync, and " +
					"Terraform re-adding it only restarts the cycle. Leave this empty for a group whose " +
					"membership Terraform manages.",
			},
		},
	}
}

func (m *model) fromAPI(group client.AuthorizationGroup) {
	m.ID = types.Int64Value(group.ID)
	m.Name = types.StringValue(group.Name)
	m.OIDCGroup = types.StringValue(group.OIDCGroup)
}

func (m model) toRequest() client.AuthorizationGroupRequest {
	return client.AuthorizationGroupRequest{
		Name:      m.Name.ValueString(),
		OIDCGroup: tfutil.StringValue(m.OIDCGroup),
	}
}

func (r *authorizationGroupResource) Create(
	ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse,
) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateAuthorizationGroup(ctx, plan.toRequest())
	if err != nil {
		resp.Diagnostics.AddError("Could not create SecObserve authorization group", err.Error())
		return
	}

	var state model
	state.fromAPI(created)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *authorizationGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	group, err := r.client.AuthorizationGroup(ctx, state.ID.ValueInt64())
	if err != nil {
		if client.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Could not read SecObserve authorization group", err.Error())
		return
	}

	var refreshed model
	refreshed.fromAPI(group)
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *authorizationGroupResource) Update(
	ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse,
) {
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

	updated, err := r.client.UpdateAuthorizationGroup(ctx, state.ID.ValueInt64(), plan.toRequest())
	if err != nil {
		resp.Diagnostics.AddError("Could not update SecObserve authorization group", err.Error())
		return
	}

	var refreshed model
	refreshed.fromAPI(updated)
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *authorizationGroupResource) Delete(
	ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse,
) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteAuthorizationGroup(ctx, state.ID.ValueInt64()); err != nil && !client.NotFound(err) {
		resp.Diagnostics.AddError("Could not delete SecObserve authorization group", err.Error())
	}
}

// ImportState accepts either the numeric id or the group's exact name.
func (r *authorizationGroupResource) ImportState(
	ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	if id, err := strconv.ParseInt(req.ID, 10, 64); err == nil {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
		return
	}

	group, err := r.client.AuthorizationGroupByName(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Could not import SecObserve authorization group",
			"Pass either the numeric id or the exact name.\n\nUnderlying error: "+err.Error(),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), group.ID)...)
}
