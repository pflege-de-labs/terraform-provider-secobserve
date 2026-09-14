// Package license_policy implements the secobserve_license_policy resource.
package license_policy //nolint:revive // the package name mirrors the Terraform resource name

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
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
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/schemacommon"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tfutil"
)

var (
	_ resource.ResourceWithConfigure   = (*licensePolicyResource)(nil)
	_ resource.ResourceWithImportState = (*licensePolicyResource)(nil)
)

// New returns the secobserve_license_policy resource.
func New() resource.Resource {
	return &licensePolicyResource{}
}

type licensePolicyResource struct {
	client *client.Client
}

type model struct {
	ID                      types.Int64  `tfsdk:"id"`
	Parent                  types.Int64  `tfsdk:"parent"`
	Name                    types.String `tfsdk:"name"`
	Description             types.String `tfsdk:"description"`
	IsPublic                types.Bool   `tfsdk:"is_public"`
	IgnoreComponentTypeList types.Set    `tfsdk:"ignore_component_type_list"`

	ParentName             types.String `tfsdk:"parent_name"`
	IsParent               types.Bool   `tfsdk:"is_parent"`
	IsManager              types.Bool   `tfsdk:"is_manager"`
	HasProducts            types.Bool   `tfsdk:"has_products"`
	HasProductGroups       types.Bool   `tfsdk:"has_product_groups"`
	HasItems               types.Bool   `tfsdk:"has_items"`
	HasUsers               types.Bool   `tfsdk:"has_users"`
	HasAuthorizationGroups types.Bool   `tfsdk:"has_authorization_groups"`
}

func (r *licensePolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_license_policy"
}

func (r *licensePolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = tfutil.ResourceClient(req, resp)
}

func (r *licensePolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A license policy: a named set of allow/forbid/review rules " +
			"(`secobserve_license_policy_item`) applied to a product's or product group's components.\n\n" +
			"~> **Do not manage the seeded `Standard` policy with this resource.** It is created once, on an " +
			"empty database, as a starting point -- adopting it into Terraform is possible via import, but " +
			"there is nothing special protecting it from concurrent manual edits the way there would be for " +
			"a Terraform-only resource. Prefer declaring your own policy.\n\n" +
			"~> If the provider's token is not a superuser, creating this resource implicitly makes that " +
			"identity a manager member (`is_manager = true`); declaring a `secobserve_license_policy_member` " +
			"for the same user then fails as a duplicate. Run the provider as a superuser to avoid this.\n\n" +
			"~> **`parent` is limited to one level of nesting.** SecObserve rejects assigning a parent that " +
			"itself has a parent, assigning a parent to a policy that already has children, and assigning a " +
			"policy as its own parent -- all as a 400 at apply time; the provider does not duplicate these " +
			"checks at plan time.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{schemacommon.Int64UseStateForUnknown()},
				MarkdownDescription: "Numeric id of the license policy.",
			},
			"parent": schema.Int64Attribute{
				Optional:            true,
				Validators:          []validator.Int64{int64validator.AtLeast(1)},
				MarkdownDescription: "Id of the parent policy, if any. See the one-level-of-nesting note above.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthBetween(1, 255)},
				MarkdownDescription: "Name of the license policy. Globally unique.",
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
				MarkdownDescription: "Whether every authenticated user can see this policy, not just members.",
			},
			"ignore_component_type_list": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     setdefault.StaticValue(schemacommon.EmptyStringSet()),
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(stringvalidator.OneOf(client.PurlTypes...)),
				},
				MarkdownDescription: "Package URL types this policy skips entirely, for example `docker` to " +
					"exempt container base images. Any of " + schemacommon.MarkdownList(client.PurlTypes) + ".",
			},
			"parent_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Name of the parent policy, or empty if there is none.",
			},
			"is_parent": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether at least one other policy has this one as its parent.",
			},
			"is_manager": schema.BoolAttribute{
				Computed: true,
				MarkdownDescription: "Whether the provider's identity is an explicit manager member of this " +
					"policy. A superuser can write this resource regardless of this flag, so it reads `false` " +
					"for a superuser-created policy with no membership rows of its own.",
			},
			"has_products": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether at least one product uses this policy.",
			},
			"has_product_groups": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether at least one product group uses this policy.",
			},
			"has_items": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether this policy has at least one item.",
			},
			"has_users": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether this policy has at least one user member.",
			},
			"has_authorization_groups": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether this policy has at least one authorization group member.",
			},
		},
	}
}

func (m *model) fromAPI(ctx context.Context, policy client.LicensePolicy, diags *diag.Diagnostics) {
	m.ID = types.Int64Value(policy.ID)
	m.Parent = tfutil.Int64(policy.Parent)
	m.Name = types.StringValue(policy.Name)
	m.Description = types.StringValue(policy.Description)
	m.IsPublic = types.BoolValue(policy.IsPublic)
	m.ParentName = types.StringValue(policy.ParentName)
	m.IsParent = types.BoolValue(policy.IsParent)
	m.IsManager = types.BoolValue(policy.IsManager)
	m.HasProducts = types.BoolValue(policy.HasProducts)
	m.HasProductGroups = types.BoolValue(policy.HasProductGroups)
	m.HasItems = types.BoolValue(policy.HasItems)
	m.HasUsers = types.BoolValue(policy.HasUsers)
	m.HasAuthorizationGroups = types.BoolValue(policy.HasAuthorizationGroups)

	ignoreTypes, setDiags := types.SetValueFrom(ctx, types.StringType, policy.IgnoreComponentTypeList)
	diags.Append(setDiags...)
	m.IgnoreComponentTypeList = ignoreTypes
}

func (m model) toRequest(ctx context.Context, diags *diag.Diagnostics) client.LicensePolicyRequest {
	var ignoreTypes []string
	diags.Append(m.IgnoreComponentTypeList.ElementsAs(ctx, &ignoreTypes, false)...)

	return client.LicensePolicyRequest{
		Parent:                  tfutil.Int64Ptr(m.Parent),
		Name:                    m.Name.ValueString(),
		Description:             tfutil.StringValue(m.Description),
		IsPublic:                tfutil.BoolValue(m.IsPublic),
		IgnoreComponentTypeList: ignoreTypes,
	}
}

func (r *licensePolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := plan.toRequest(ctx, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateLicensePolicy(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError("Could not create SecObserve license policy", err.Error())
		return
	}

	var state model
	state.fromAPI(ctx, created, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *licensePolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy, err := r.client.LicensePolicy(ctx, state.ID.ValueInt64())
	if err != nil {
		if client.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Could not read SecObserve license policy", err.Error())
		return
	}

	var refreshed model
	refreshed.fromAPI(ctx, policy, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *licensePolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
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

	request := plan.toRequest(ctx, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateLicensePolicy(ctx, state.ID.ValueInt64(), request)
	if err != nil {
		resp.Diagnostics.AddError("Could not update SecObserve license policy", err.Error())
		return
	}

	var refreshed model
	refreshed.fromAPI(ctx, updated, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *licensePolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteLicensePolicy(ctx, state.ID.ValueInt64()); err != nil && !client.NotFound(err) {
		resp.Diagnostics.AddError("Could not delete SecObserve license policy", err.Error())
	}
}

// ImportState accepts either the numeric id or the policy's exact name.
func (r *licensePolicyResource) ImportState(
	ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	if id, err := strconv.ParseInt(req.ID, 10, 64); err == nil {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
		return
	}

	policy, err := r.client.LicensePolicyByName(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Could not import SecObserve license policy",
			"Pass either the numeric id or the exact name.\n\nUnderlying error: "+err.Error(),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), policy.ID)...)
}
