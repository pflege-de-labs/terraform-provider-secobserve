// Package authorization_group_member implements the
// secobserve_authorization_group_member resource.
package authorization_group_member //nolint:revive // the package name mirrors the Terraform resource name

import (
	"context"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/schemacommon"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tfutil"
)

var (
	_ resource.ResourceWithConfigure   = (*memberResource)(nil)
	_ resource.ResourceWithImportState = (*memberResource)(nil)
)

// New returns the secobserve_authorization_group_member resource.
func New() resource.Resource {
	return &memberResource{}
}

type memberResource struct {
	client *client.Client
}

type model struct {
	ID                 types.Int64 `tfsdk:"id"`
	AuthorizationGroup types.Int64 `tfsdk:"authorization_group"`
	User               types.Int64 `tfsdk:"user"`
	IsManager          types.Bool  `tfsdk:"is_manager"`
}

func (r *memberResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_authorization_group_member"
}

func (r *memberResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = tfutil.ResourceClient(req, resp)
}

func (r *memberResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Adds a user to an authorization group.\n\n" +
			"~> If the group has `oidc_group` set, do not manage its membership here: SecObserve's " +
			"login-time group sync deletes any membership of an OIDC-mapped group that was not granted by " +
			"the IdP, so a Terraform-managed membership is removed at the member's next login. The " +
			"provider warns when this happens.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{schemacommon.Int64UseStateForUnknown()},
				MarkdownDescription: "Numeric id of the membership.",
			},
			"authorization_group": schema.Int64Attribute{
				Required:      true,
				Validators:    []validator.Int64{int64validator.AtLeast(1)},
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
				MarkdownDescription: "Id of the authorization group. Immutable: SecObserve rejects moving a " +
					"membership, so a change replaces it.",
			},
			"user": schema.Int64Attribute{
				Required:      true,
				Validators:    []validator.Int64{int64validator.AtLeast(1)},
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
				MarkdownDescription: "Id of the user. Use the `secobserve_user` data source to resolve a " +
					"username. Immutable: a change replaces the membership.",
			},
			"is_manager": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
				MarkdownDescription: "Whether this member can manage the group itself: edit it, and add or " +
					"remove other members.",
			},
		},
	}
}

func (m *model) fromAPI(member client.AuthorizationGroupMember) {
	m.ID = types.Int64Value(member.ID)
	m.AuthorizationGroup = types.Int64Value(member.AuthorizationGroup)
	m.User = types.Int64Value(member.User)
	m.IsManager = types.BoolValue(member.IsManager)
}

func (r *memberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateAuthorizationGroupMember(ctx, client.AuthorizationGroupMemberRequest{
		AuthorizationGroup: plan.AuthorizationGroup.ValueInt64(),
		User:               plan.User.ValueInt64(),
		IsManager:          tfutil.BoolValue(plan.IsManager),
	})
	if err != nil {
		resp.Diagnostics.AddError("Could not create SecObserve authorization group member", err.Error())
		return
	}

	r.warnIfOIDCManaged(ctx, plan.AuthorizationGroup.ValueInt64(), &resp.Diagnostics)

	var state model
	state.fromAPI(created)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *memberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	member, err := r.client.AuthorizationGroupMember(ctx, state.ID.ValueInt64())
	if err != nil {
		if client.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Could not read SecObserve authorization group member", err.Error())
		return
	}

	var refreshed model
	refreshed.fromAPI(member)
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

// Update only ever changes is_manager: the other two attributes force a
// replace.
func (r *memberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
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

	updated, err := r.client.UpdateAuthorizationGroupMemberIsManager(
		ctx, state.ID.ValueInt64(), tfutil.BoolValue(plan.IsManager))
	if err != nil {
		resp.Diagnostics.AddError("Could not update SecObserve authorization group member", err.Error())
		return
	}

	r.warnIfOIDCManaged(ctx, plan.AuthorizationGroup.ValueInt64(), &resp.Diagnostics)

	var refreshed model
	refreshed.fromAPI(updated)
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *memberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteAuthorizationGroupMember(ctx, state.ID.ValueInt64()); err != nil && !client.NotFound(err) {
		resp.Diagnostics.AddError("Could not delete SecObserve authorization group member", err.Error())
	}
}

// warnIfOIDCManaged reads the target group back and warns when its
// oidc_group is set, since login-time sync will delete this membership. A
// failure to read the group is not itself an error worth failing the apply
// over -- the membership write already succeeded -- so it is logged as a
// second, more generic warning instead.
func (r *memberResource) warnIfOIDCManaged(ctx context.Context, groupID int64, diags *diag.Diagnostics) {
	group, err := r.client.AuthorizationGroup(ctx, groupID)
	if err != nil {
		diags.AddWarning(
			"Could not verify whether the authorization group is OIDC-managed",
			"Could not read authorization group "+strconv.FormatInt(groupID, 10)+" to check its oidc_group "+
				"attribute after writing this membership.\n\nUnderlying error: "+err.Error(),
		)
		return
	}
	if group.OIDCGroup == "" {
		return
	}
	diags.AddWarning(
		"Membership of an OIDC-managed authorization group",
		"Authorization group "+strconv.FormatInt(groupID, 10)+" (\""+group.Name+"\") has oidc_group = \""+
			group.OIDCGroup+"\" set. SecObserve's login-time group synchronization will delete this "+
			"membership the next time the member logs in via OIDC, because it was not granted by the "+
			"identity provider. Manage membership of this group through the IdP instead.",
	)
}

// ImportState takes "<authorization group id>/<user id>", the membership's
// natural key.
func (r *memberResource) ImportState(
	ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	groupID, userID, found := strings.Cut(req.ID, "/")
	if !found {
		resp.Diagnostics.AddError(
			"Invalid import id",
			"Import as \"<authorization group id>/<user id>\", for example \"3/12\".",
		)
		return
	}

	parsedGroup, groupErr := strconv.ParseInt(groupID, 10, 64)
	parsedUser, userErr := strconv.ParseInt(userID, 10, 64)
	if groupErr != nil || userErr != nil {
		resp.Diagnostics.AddError(
			"Invalid import id",
			"Both parts of \"<authorization group id>/<user id>\" must be numeric.",
		)
		return
	}

	member, err := r.client.FindAuthorizationGroupMember(ctx, parsedGroup, parsedUser)
	if err != nil {
		resp.Diagnostics.AddError("Could not import SecObserve authorization group member", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), member.ID)...)
}
