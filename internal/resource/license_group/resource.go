// Package license_group implements the secobserve_license_group resource.
package license_group //nolint:revive // the package name mirrors the Terraform resource name

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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/schemacommon"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tfutil"
)

var (
	_ resource.ResourceWithConfigure   = (*licenseGroupResource)(nil)
	_ resource.ResourceWithImportState = (*licenseGroupResource)(nil)
)

// New returns the secobserve_license_group resource.
func New() resource.Resource {
	return &licenseGroupResource{}
}

type licenseGroupResource struct {
	client *client.Client
}

type model struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	IsPublic    types.Bool   `tfsdk:"is_public"`
	Licenses    types.Set    `tfsdk:"licenses"`

	IsManager              types.Bool `tfsdk:"is_manager"`
	IsInLicensePolicy      types.Bool `tfsdk:"is_in_license_policy"`
	HasLicenses            types.Bool `tfsdk:"has_licenses"`
	HasUsers               types.Bool `tfsdk:"has_users"`
	HasAuthorizationGroups types.Bool `tfsdk:"has_authorization_groups"`
}

func (r *licenseGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_license_group"
}

func (r *licenseGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = tfutil.ResourceClient(req, resp)
}

func (r *licenseGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A named collection of SPDX licenses, referenced from license policy items.\n\n" +
			"~> **Do not manage a `(ScanCode LicenseDB)` group with this resource.** SecObserve reimports " +
			"those nightly, clearing and rebuilding their license membership every time " +
			"(`feature_license_management`, enabled by default) -- any Terraform-managed `licenses` set on " +
			"one is lost within 24h. Reference them read-only via the `secobserve_license_group` data " +
			"source instead, or declare your own group.\n\n" +
			"~> If the provider's token is not a superuser, creating this resource implicitly makes that " +
			"identity a manager member (`is_manager = true`); declaring a `secobserve_license_group_member` " +
			"for the same user then fails as a duplicate. Run the provider as a superuser to avoid this.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{schemacommon.Int64UseStateForUnknown()},
				MarkdownDescription: "Numeric id of the license group.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthBetween(1, 255)},
				MarkdownDescription: "Name of the license group. Globally unique.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				Validators:          []validator.String{stringvalidator.LengthAtMost(2048)},
				MarkdownDescription: "Free-text description.",
			},
			"is_public": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether every authenticated user can see this group, not just members.",
			},
			"licenses": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.Int64Type,
				Default:     setdefault.StaticValue(schemacommon.EmptyInt64Set()),
				MarkdownDescription: "Ids of the licenses in this group. Use the `secobserve_license` data " +
					"source to resolve an SPDX id to the numeric id required here.\n\n" +
					"~> Reconciled through SecObserve's `add_license`/`remove_license` actions, since the " +
					"license group API never accepts this list directly. An empty set is a real, " +
					"empty group, not \"unmanaged\".",
			},
			"is_manager": schema.BoolAttribute{
				Computed: true,
				MarkdownDescription: "Whether the provider's identity is an explicit manager member of this " +
					"group. A superuser can write this resource regardless of this flag, so it reads `false` " +
					"for a superuser-created group with no membership rows of its own.",
			},
			"is_in_license_policy": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether any license policy item references this group.",
			},
			"has_licenses": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether this group contains at least one license.",
			},
			"has_users": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether this group has at least one user member.",
			},
			"has_authorization_groups": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether this group has at least one authorization group member.",
			},
		},
	}
}

func (m *model) fromAPI(ctx context.Context, group client.LicenseGroup, licenses []client.License, diags *diag.Diagnostics) {
	m.ID = types.Int64Value(group.ID)
	m.Name = types.StringValue(group.Name)
	m.Description = types.StringValue(group.Description)
	m.IsPublic = types.BoolValue(group.IsPublic)
	m.IsManager = types.BoolValue(group.IsManager)
	m.IsInLicensePolicy = types.BoolValue(group.IsInLicensePolicy)
	m.HasLicenses = types.BoolValue(group.HasLicenses)
	m.HasUsers = types.BoolValue(group.HasUsers)
	m.HasAuthorizationGroups = types.BoolValue(group.HasAuthorizationGroups)

	ids := make([]int64, 0, len(licenses))
	for _, license := range licenses {
		ids = append(ids, license.ID)
	}
	set, setDiags := types.SetValueFrom(ctx, types.Int64Type, ids)
	diags.Append(setDiags...)
	m.Licenses = set
}

func (m model) toRequest() client.LicenseGroupRequest {
	return client.LicenseGroupRequest{
		Name:        m.Name.ValueString(),
		Description: tfutil.StringValue(m.Description),
		IsPublic:    tfutil.BoolValue(m.IsPublic),
	}
}

func (r *licenseGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateLicenseGroup(ctx, plan.toRequest())
	if err != nil {
		resp.Diagnostics.AddError("Could not create SecObserve license group", err.Error())
		return
	}

	var wantedIDs []int64
	resp.Diagnostics.Append(plan.Licenses.ElementsAs(ctx, &wantedIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	for _, licenseID := range wantedIDs {
		if err := r.client.AddLicenseToGroup(ctx, created.ID, licenseID); err != nil {
			resp.Diagnostics.AddError("Could not add license to SecObserve license group", err.Error())
			return
		}
	}

	r.readInto(ctx, created.ID, &resp.State, &resp.Diagnostics)
}

func (r *licenseGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	group, err := r.client.LicenseGroup(ctx, state.ID.ValueInt64())
	if err != nil {
		if client.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Could not read SecObserve license group", err.Error())
		return
	}

	licenses, err := r.client.ListLicensesInGroup(ctx, group.ID)
	if err != nil {
		resp.Diagnostics.AddError("Could not list licenses in SecObserve license group", err.Error())
		return
	}

	var refreshed model
	refreshed.fromAPI(ctx, group, licenses, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

// readInto re-reads a license group plus its licenses and writes both into
// state, shared by Create and Update after they finish reconciling the
// licenses set.
func (r *licenseGroupResource) readInto(ctx context.Context, id int64, state *tfsdk.State, diags *diag.Diagnostics) {
	group, err := r.client.LicenseGroup(ctx, id)
	if err != nil {
		diags.AddError("Could not read SecObserve license group", err.Error())
		return
	}
	licenses, err := r.client.ListLicensesInGroup(ctx, id)
	if err != nil {
		diags.AddError("Could not list licenses in SecObserve license group", err.Error())
		return
	}

	var result model
	result.fromAPI(ctx, group, licenses, diags)
	diags.Append(state.Set(ctx, result)...)
}

func (r *licenseGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
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

	id := state.ID.ValueInt64()
	if _, err := r.client.UpdateLicenseGroup(ctx, id, plan.toRequest()); err != nil {
		resp.Diagnostics.AddError("Could not update SecObserve license group", err.Error())
		return
	}

	var currentIDs, wantedIDs []int64
	resp.Diagnostics.Append(state.Licenses.ElementsAs(ctx, &currentIDs, false)...)
	resp.Diagnostics.Append(plan.Licenses.ElementsAs(ctx, &wantedIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	wanted := toSet(wantedIDs)
	current := toSet(currentIDs)

	for licenseID := range current {
		if !wanted[licenseID] {
			if err := r.client.RemoveLicenseFromGroup(ctx, id, licenseID); err != nil {
				resp.Diagnostics.AddError("Could not remove license from SecObserve license group", err.Error())
				return
			}
		}
	}
	for licenseID := range wanted {
		if !current[licenseID] {
			if err := r.client.AddLicenseToGroup(ctx, id, licenseID); err != nil {
				resp.Diagnostics.AddError("Could not add license to SecObserve license group", err.Error())
				return
			}
		}
	}

	r.readInto(ctx, id, &resp.State, &resp.Diagnostics)
}

func (r *licenseGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteLicenseGroup(ctx, state.ID.ValueInt64()); err != nil && !client.NotFound(err) {
		resp.Diagnostics.AddError("Could not delete SecObserve license group", err.Error())
	}
}

// ImportState accepts either the numeric id or the group's exact name.
func (r *licenseGroupResource) ImportState(
	ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	if id, err := strconv.ParseInt(req.ID, 10, 64); err == nil {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
		return
	}

	group, err := r.client.LicenseGroupByName(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Could not import SecObserve license group",
			"Pass either the numeric id or the exact name.\n\nUnderlying error: "+err.Error(),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), group.ID)...)
}

func toSet(ids []int64) map[int64]bool {
	set := make(map[int64]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set
}
