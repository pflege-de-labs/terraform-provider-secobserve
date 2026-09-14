// Package user implements the secobserve_user resource.
package user

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/schemacommon"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tfutil"
)

var (
	_ resource.ResourceWithConfigure   = (*userResource)(nil)
	_ resource.ResourceWithImportState = (*userResource)(nil)
)

// New returns the secobserve_user resource.
func New() resource.Resource {
	return &userResource{}
}

type userResource struct {
	client *client.Client
}

type model struct {
	ID          types.Int64  `tfsdk:"id"`
	Username    types.String `tfsdk:"username"`
	FirstName   types.String `tfsdk:"first_name"`
	LastName    types.String `tfsdk:"last_name"`
	FullName    types.String `tfsdk:"full_name"`
	Email       types.String `tfsdk:"email"`
	IsActive    types.Bool   `tfsdk:"is_active"`
	IsSuperuser types.Bool   `tfsdk:"is_superuser"`
	IsExternal  types.Bool   `tfsdk:"is_external"`
	IsOIDCUser  types.Bool   `tfsdk:"is_oidc_user"`
}

func (r *userResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *userResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = tfutil.ResourceClient(req, resp)
}

func (r *userResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A local SecObserve user, manageable by a superuser through the API.\n\n" +
			"~> Users created this way have **no usable password**: the write endpoint this resource uses " +
			"has no password field. Set one out of band with `PATCH /api/users/{id}/change_password/`, " +
			"which requires the caller's own current password and so cannot be done by this provider, or " +
			"have the user authenticate via OIDC instead.\n\n" +
			"On an OIDC-enabled instance, prefer the `secobserve_user` data source to *reference* users " +
			"provisioned just-in-time at login, rather than this resource: SecObserve overwrites `email`, " +
			"`full_name`, `first_name` and `last_name` from the identity token on every OIDC login, fighting " +
			"anything Terraform sets here. `is_active`, `is_superuser` and `is_external` are not touched by " +
			"login and remain safely Terraform-managed even for an OIDC user.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{schemacommon.Int64UseStateForUnknown()},
				MarkdownDescription: "Numeric id of the user.",
			},
			"username": schema.StringAttribute{
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthBetween(1, 150)},
				MarkdownDescription: "Username. Globally unique.",
			},
			"first_name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				MarkdownDescription: "First name.",
			},
			"last_name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				MarkdownDescription: "Last name.",
			},
			"full_name": schema.StringAttribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "Display name.\n\n" +
					"~> SecObserve recomputes this from `first_name` and `last_name` on every write whenever " +
					"either is non-empty, overriding whatever is set here. It is only honored as given when " +
					"both are empty. Leave it unset unless you are deliberately setting a display name for a " +
					"user with no first or last name.",
			},
			"email": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				MarkdownDescription: "Email address.",
			},
			"is_active": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether the user can authenticate.",
			},
			"is_superuser": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
				MarkdownDescription: "Whether the user has superuser privileges: full read/write access " +
					"regardless of product membership, and the ability to manage other users, general rules, " +
					"instance settings and periodic tasks.",
			},
			"is_external": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
				MarkdownDescription: "Whether the user is external. External users cannot create products " +
					"or product groups.",
			},
			"is_oidc_user": schema.BoolAttribute{
				Computed: true,
				MarkdownDescription: "Whether SecObserve considers this an OIDC-provisioned user. Read-only: " +
					"set by SecObserve itself on first OIDC login, never by this resource.",
			},
		},
	}
}

func (m *model) fromAPI(user client.User) {
	m.ID = types.Int64Value(user.ID)
	m.Username = types.StringValue(user.Username)
	m.FirstName = types.StringValue(user.FirstName)
	m.LastName = types.StringValue(user.LastName)
	m.FullName = types.StringValue(user.FullName)
	m.Email = types.StringValue(user.Email)
	m.IsActive = types.BoolValue(user.IsActive)
	m.IsSuperuser = types.BoolValue(user.IsSuperuser)
	m.IsExternal = types.BoolValue(user.IsExternal)
	m.IsOIDCUser = types.BoolValue(user.IsOIDCUser)
}

func (m model) toRequest() client.UserRequest {
	return client.UserRequest{
		Username:    m.Username.ValueString(),
		FirstName:   tfutil.StringValue(m.FirstName),
		LastName:    tfutil.StringValue(m.LastName),
		FullName:    tfutil.StringValue(m.FullName),
		Email:       tfutil.StringValue(m.Email),
		IsActive:    tfutil.BoolValue(m.IsActive),
		IsSuperuser: tfutil.BoolValue(m.IsSuperuser),
		IsExternal:  tfutil.BoolValue(m.IsExternal),
	}
}

// warnIfOIDCClaimsSet reports that SecObserve will overwrite claim-owned
// fields on this user's next OIDC login, once the API confirms the user is
// already OIDC-provisioned.
func warnIfOIDCClaimsSet(m model, diags *diag.Diagnostics) {
	if !m.IsOIDCUser.ValueBool() {
		return
	}
	if tfutil.StringValue(m.Email) == "" && tfutil.StringValue(m.FullName) == "" &&
		tfutil.StringValue(m.FirstName) == "" && tfutil.StringValue(m.LastName) == "" {
		return
	}
	diags.AddWarning(
		"Managing claim-owned attributes of an OIDC user",
		"User \""+m.Username.ValueString()+"\" is OIDC-provisioned (is_oidc_user = true). SecObserve "+
			"overwrites email, full_name, first_name and last_name from the identity token on every OIDC "+
			"login, so any value Terraform sets for them here will be reverted at the user's next login. "+
			"Only is_active, is_superuser and is_external are safely managed for an OIDC user.",
	)
}

func (r *userResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateUser(ctx, plan.toRequest())
	if err != nil {
		resp.Diagnostics.AddError("Could not create SecObserve user", err.Error())
		return
	}

	resp.Diagnostics.AddWarning(
		"User has no usable password",
		"SecObserve's user-create endpoint has no password field. \""+created.Username+"\" cannot log in "+
			"with a password until one is set via PATCH /api/users/{id}/change_password/ (which requires "+
			"the caller's own current password) or the user authenticates via OIDC.",
	)

	var state model
	state.fromAPI(created)
	warnIfOIDCClaimsSet(state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *userResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := r.client.User(ctx, state.ID.ValueInt64())
	if err != nil {
		if client.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Could not read SecObserve user", err.Error())
		return
	}

	var refreshed model
	refreshed.fromAPI(user)
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *userResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
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

	updated, err := r.client.UpdateUser(ctx, state.ID.ValueInt64(), plan.toRequest())
	if err != nil {
		resp.Diagnostics.AddError("Could not update SecObserve user", err.Error())
		return
	}

	var refreshed model
	refreshed.fromAPI(updated)
	warnIfOIDCClaimsSet(refreshed, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *userResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteUser(ctx, state.ID.ValueInt64())
	if err != nil && !client.NotFound(err) {
		resp.Diagnostics.AddError(
			"Could not delete SecObserve user",
			"SecObserve refuses to let a user delete themselves. If this is the identity the provider "+
				"authenticates as, delete it out of band instead.\n\nUnderlying error: "+err.Error(),
		)
	}
}

// ImportState accepts either the numeric id or the user's exact username.
func (r *userResource) ImportState(
	ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	if id, err := strconv.ParseInt(req.ID, 10, 64); err == nil {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
		return
	}

	user, err := r.client.UserByUsername(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Could not import SecObserve user",
			"Pass either the numeric id or the exact username.\n\nUnderlying error: "+err.Error(),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), user.ID)...)
}
