// Package license_policy_member implements the secobserve_license_policy_member resource.
package license_policy_member //nolint:revive // the package name mirrors the Terraform resource name

import (
	"context"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
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

// New returns the secobserve_license_policy_member resource.
func New() resource.Resource {
	return &memberResource{}
}

type memberResource struct {
	client *client.Client
}

type model struct {
	ID            types.Int64 `tfsdk:"id"`
	LicensePolicy types.Int64 `tfsdk:"license_policy"`
	User          types.Int64 `tfsdk:"user"`
	IsManager     types.Bool  `tfsdk:"is_manager"`
}

func (r *memberResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_license_policy_member"
}

func (r *memberResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = tfutil.ResourceClient(req, resp)
}

func (r *memberResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Grants a user manage rights on a license policy: edit it, and add or remove " +
			"other members and items.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{schemacommon.Int64UseStateForUnknown()},
				MarkdownDescription: "Numeric id of the membership.",
			},
			"license_policy": schema.Int64Attribute{
				Required:      true,
				Validators:    []validator.Int64{int64validator.AtLeast(1)},
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
				MarkdownDescription: "Id of the license policy. Immutable: SecObserve rejects moving a " +
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
				MarkdownDescription: "Whether this member can manage the policy itself: edit it, and add or " +
					"remove other members and items.",
			},
		},
	}
}

func (m *model) fromAPI(member client.LicensePolicyMember) {
	m.ID = types.Int64Value(member.ID)
	m.LicensePolicy = types.Int64Value(member.LicensePolicy)
	m.User = types.Int64Value(member.User)
	m.IsManager = types.BoolValue(member.IsManager)
}

func (r *memberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateLicensePolicyMember(ctx, client.LicensePolicyMemberRequest{
		LicensePolicy: plan.LicensePolicy.ValueInt64(),
		User:          plan.User.ValueInt64(),
		IsManager:     tfutil.BoolValue(plan.IsManager),
	})
	if err != nil {
		resp.Diagnostics.AddError("Could not create SecObserve license policy member", err.Error())
		return
	}

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

	member, err := r.client.LicensePolicyMember(ctx, state.ID.ValueInt64())
	if err != nil {
		if client.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Could not read SecObserve license policy member", err.Error())
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

	updated, err := r.client.UpdateLicensePolicyMemberIsManager(
		ctx, state.ID.ValueInt64(), tfutil.BoolValue(plan.IsManager))
	if err != nil {
		resp.Diagnostics.AddError("Could not update SecObserve license policy member", err.Error())
		return
	}

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

	if err := r.client.DeleteLicensePolicyMember(ctx, state.ID.ValueInt64()); err != nil && !client.NotFound(err) {
		resp.Diagnostics.AddError("Could not delete SecObserve license policy member", err.Error())
	}
}

// ImportState takes "<license policy id>/<user id>", the membership's
// natural key.
func (r *memberResource) ImportState(
	ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	policyID, userID, found := strings.Cut(req.ID, "/")
	if !found {
		resp.Diagnostics.AddError(
			"Invalid import id",
			"Import as \"<license policy id>/<user id>\", for example \"3/12\".",
		)
		return
	}

	parsedPolicy, policyErr := strconv.ParseInt(policyID, 10, 64)
	parsedUser, userErr := strconv.ParseInt(userID, 10, 64)
	if policyErr != nil || userErr != nil {
		resp.Diagnostics.AddError(
			"Invalid import id",
			"Both parts of \"<license policy id>/<user id>\" must be numeric.",
		)
		return
	}

	member, err := r.client.FindLicensePolicyMember(ctx, parsedPolicy, parsedUser)
	if err != nil {
		resp.Diagnostics.AddError("Could not import SecObserve license policy member", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), member.ID)...)
}
