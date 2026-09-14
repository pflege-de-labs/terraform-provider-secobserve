// Package product_authorization_group_member implements the
// secobserve_product_authorization_group_member resource.
package product_authorization_group_member

import (
	"context"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/schemacommon"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tfutil"
)

var (
	_ resource.ResourceWithConfigure   = (*groupMemberResource)(nil)
	_ resource.ResourceWithImportState = (*groupMemberResource)(nil)
)

// New returns the secobserve_product_authorization_group_member resource.
func New() resource.Resource {
	return &groupMemberResource{}
}

type groupMemberResource struct {
	client *client.Client
}

type model struct {
	ID                 types.Int64  `tfsdk:"id"`
	Product            types.Int64  `tfsdk:"product"`
	AuthorizationGroup types.Int64  `tfsdk:"authorization_group"`
	Role               types.String `tfsdk:"role"`
}

func (r *groupMemberResource) Metadata(
	_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_product_authorization_group_member"
}

func (r *groupMemberResource) Configure(
	_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse,
) {
	r.client = tfutil.ResourceClient(req, resp)
}

func (r *groupMemberResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := map[string]schema.Attribute{
		"id": schema.Int64Attribute{
			Computed:            true,
			PlanModifiers:       []planmodifier.Int64{schemacommon.Int64UseStateForUnknown()},
			MarkdownDescription: "Numeric id of the membership.",
		},
		"product": schema.Int64Attribute{
			Required:      true,
			Validators:    []validator.Int64{int64validator.AtLeast(1)},
			PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			MarkdownDescription: "Id of the product or product group. Immutable: SecObserve rejects moving a " +
				"membership, so a change replaces it.",
		},
		"authorization_group": schema.Int64Attribute{
			Required:      true,
			Validators:    []validator.Int64{int64validator.AtLeast(1)},
			PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			MarkdownDescription: "Id of the authorization group. Immutable: a change replaces the " +
				"membership.",
		},
	}
	schemacommon.AddRole(attributes, "authorization group")

	resp.Schema = schema.Schema{
		MarkdownDescription: "Grants an authorization group a role on a product or a product group. The same " +
			"endpoint serves both, so pass a product group's id in `product` to grant a role there.\n\n" +
			"~> While the group is a designated assessment approver of the product, SecObserve refuses to " +
			"lower its role below `Writer`.",
		Attributes: attributes,
	}
}

func (r *groupMemberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateProductAuthorizationGroupMember(
		ctx, client.ProductAuthorizationGroupMemberRequest{
			Product:            plan.Product.ValueInt64(),
			AuthorizationGroup: plan.AuthorizationGroup.ValueInt64(),
			Role:               schemacommon.RoleValue(plan.Role.ValueString()),
		})
	if err != nil {
		resp.Diagnostics.AddError("Could not create SecObserve product authorization group member", err.Error())
		return
	}

	var state model
	state.fromAPI(created)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *groupMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	member, err := r.client.ProductAuthorizationGroupMember(ctx, state.ID.ValueInt64())
	if err != nil {
		if client.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Could not read SecObserve product authorization group member", err.Error())
		return
	}

	var refreshed model
	refreshed.fromAPI(member)
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

// Update only ever changes the role: the other two attributes force a replace.
func (r *groupMemberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
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

	updated, err := r.client.UpdateProductAuthorizationGroupMemberRole(
		ctx, state.ID.ValueInt64(), schemacommon.RoleValue(plan.Role.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Could not update SecObserve product authorization group member",
			"SecObserve refuses to lower the role below Writer while the group is a designated assessment "+
				"approver of the product.\n\nUnderlying error: "+err.Error(),
		)
		return
	}

	var refreshed model
	refreshed.fromAPI(updated)
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *groupMemberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteProductAuthorizationGroupMember(ctx, state.ID.ValueInt64())
	if err != nil && !client.NotFound(err) {
		resp.Diagnostics.AddError("Could not delete SecObserve product authorization group member", err.Error())
	}
}

func (m *model) fromAPI(member client.ProductAuthorizationGroupMember) {
	m.ID = types.Int64Value(member.ID)
	m.Product = types.Int64Value(member.Product)
	m.AuthorizationGroup = types.Int64Value(member.AuthorizationGroup)
	m.Role = types.StringValue(schemacommon.RoleName(member.Role))
}

// ImportState takes "<product id>/<authorization group id>".
func (r *groupMemberResource) ImportState(
	ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	productID, groupID, found := strings.Cut(req.ID, "/")
	if !found {
		resp.Diagnostics.AddError(
			"Invalid import id",
			"Import as \"<product id>/<authorization group id>\", for example \"12/3\".",
		)
		return
	}

	parsedProduct, productErr := strconv.ParseInt(productID, 10, 64)
	parsedGroup, groupErr := strconv.ParseInt(groupID, 10, 64)
	if productErr != nil || groupErr != nil {
		resp.Diagnostics.AddError(
			"Invalid import id",
			"Both parts of \"<product id>/<authorization group id>\" must be numeric.",
		)
		return
	}

	member, err := r.client.FindProductAuthorizationGroupMember(ctx, parsedProduct, parsedGroup)
	if err != nil {
		resp.Diagnostics.AddError("Could not import SecObserve product authorization group member", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), member.ID)...)
}
