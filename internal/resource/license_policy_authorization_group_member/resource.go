// Package license_policy_authorization_group_member implements the
// secobserve_license_policy_authorization_group_member resource.
package license_policy_authorization_group_member //nolint:revive // the package name mirrors the Terraform resource name

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

// New returns the secobserve_license_policy_authorization_group_member
// resource.
func New() resource.Resource {
	return &memberResource{}
}

type memberResource struct {
	client *client.Client
}

type model struct {
	ID                 types.Int64 `tfsdk:"id"`
	LicensePolicy      types.Int64 `tfsdk:"license_policy"`
	AuthorizationGroup types.Int64 `tfsdk:"authorization_group"`
	IsManager          types.Bool  `tfsdk:"is_manager"`
}

func (r *memberResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_license_policy_authorization_group_member"
}

func (r *memberResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = tfutil.ResourceClient(req, resp)
}

func (r *memberResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Grants every member of an authorization group manage rights on a license " +
			"policy, instead of one membership per user.",
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
			"authorization_group": schema.Int64Attribute{
				Required:            true,
				Validators:          []validator.Int64{int64validator.AtLeast(1)},
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
				MarkdownDescription: "Id of the authorization group. Immutable: a change replaces the membership.",
			},
			"is_manager": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
				MarkdownDescription: "Whether members of this authorization group can manage the license " +
					"policy itself: edit it, and add or remove other members and items.",
			},
		},
	}
}

func (m *model) fromAPI(member client.LicensePolicyAuthorizationGroupMember) {
	m.ID = types.Int64Value(member.ID)
	m.LicensePolicy = types.Int64Value(member.LicensePolicy)
	m.AuthorizationGroup = types.Int64Value(member.AuthorizationGroup)
	m.IsManager = types.BoolValue(member.IsManager)
}

func (r *memberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateLicensePolicyAuthorizationGroupMember(ctx, client.LicensePolicyAuthorizationGroupMemberRequest{
		LicensePolicy:      plan.LicensePolicy.ValueInt64(),
		AuthorizationGroup: plan.AuthorizationGroup.ValueInt64(),
		IsManager:          tfutil.BoolValue(plan.IsManager),
	})
	if err != nil {
		resp.Diagnostics.AddError("Could not create SecObserve license policy authorization group member", err.Error())
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

	member, err := r.client.LicensePolicyAuthorizationGroupMember(ctx, state.ID.ValueInt64())
	if err != nil {
		if client.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Could not read SecObserve license policy authorization group member", err.Error())
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

	updated, err := r.client.UpdateLicensePolicyAuthorizationGroupMemberIsManager(
		ctx, state.ID.ValueInt64(), tfutil.BoolValue(plan.IsManager))
	if err != nil {
		resp.Diagnostics.AddError("Could not update SecObserve license policy authorization group member", err.Error())
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

	if err := r.client.DeleteLicensePolicyAuthorizationGroupMember(ctx, state.ID.ValueInt64()); err != nil && !client.NotFound(err) {
		resp.Diagnostics.AddError("Could not delete SecObserve license policy authorization group member", err.Error())
	}
}

// ImportState takes "<license policy id>/<authorization group id>", the
// membership's natural key.
func (r *memberResource) ImportState(
	ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	policyID, authGroupID, found := strings.Cut(req.ID, "/")
	if !found {
		resp.Diagnostics.AddError(
			"Invalid import id",
			"Import as \"<license policy id>/<authorization group id>\", for example \"3/12\".",
		)
		return
	}

	parsedPolicy, policyErr := strconv.ParseInt(policyID, 10, 64)
	parsedAuthGroup, authGroupErr := strconv.ParseInt(authGroupID, 10, 64)
	if policyErr != nil || authGroupErr != nil {
		resp.Diagnostics.AddError(
			"Invalid import id",
			"Both parts of \"<license policy id>/<authorization group id>\" must be numeric.",
		)
		return
	}

	member, err := r.client.FindLicensePolicyAuthorizationGroupMember(ctx, parsedPolicy, parsedAuthGroup)
	if err != nil {
		resp.Diagnostics.AddError("Could not import SecObserve license policy authorization group member", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), member.ID)...)
}
