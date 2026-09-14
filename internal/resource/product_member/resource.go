// Package product_member implements the secobserve_product_member resource.
package product_member

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
	_ resource.ResourceWithConfigure   = (*productMemberResource)(nil)
	_ resource.ResourceWithImportState = (*productMemberResource)(nil)
)

// New returns the secobserve_product_member resource.
func New() resource.Resource {
	return &productMemberResource{}
}

type productMemberResource struct {
	client *client.Client
}

type model struct {
	ID      types.Int64  `tfsdk:"id"`
	Product types.Int64  `tfsdk:"product"`
	User    types.Int64  `tfsdk:"user"`
	Role    types.String `tfsdk:"role"`
}

func (r *productMemberResource) Metadata(
	_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_product_member"
}

func (r *productMemberResource) Configure(
	_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse,
) {
	r.client = tfutil.ResourceClient(req, resp)
}

func (r *productMemberResource) Schema(
	_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse,
) {
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
		"user": schema.Int64Attribute{
			Required:      true,
			Validators:    []validator.Int64{int64validator.AtLeast(1)},
			PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			MarkdownDescription: "Id of the user. Use the `secobserve_user` data source to resolve a " +
				"username. Immutable: a change replaces the membership.",
		},
	}
	schemacommon.AddRole(attributes, "user")

	resp.Schema = schema.Schema{
		MarkdownDescription: "Grants a user a role on a product or a product group. The same endpoint serves " +
			"both, so pass a product group's id in `product` to grant a role there.\n\n" +
			"~> Whoever creates a product becomes an `Owner` member of it automatically. Declaring a " +
			"membership for the identity the provider authenticates as therefore fails as a duplicate.",
		Attributes: attributes,
	}
}

func (r *productMemberResource) Create(
	ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse,
) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateProductMember(ctx, client.ProductMemberRequest{
		Product: plan.Product.ValueInt64(),
		User:    plan.User.ValueInt64(),
		Role:    schemacommon.RoleValue(plan.Role.ValueString()),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Could not create SecObserve product member",
			"If this reports a duplicate, note that the creator of a product is already an Owner member of "+
				"it.\n\nUnderlying error: "+err.Error(),
		)
		return
	}

	var state model
	state.fromAPI(created)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *productMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	member, err := r.client.ProductMember(ctx, state.ID.ValueInt64())
	if err != nil {
		if client.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Could not read SecObserve product member", err.Error())
		return
	}

	var refreshed model
	refreshed.fromAPI(member)
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

// Update only ever changes the role: product and user force a replace.
func (r *productMemberResource) Update(
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

	updated, err := r.client.UpdateProductMemberRole(
		ctx, state.ID.ValueInt64(), schemacommon.RoleValue(plan.Role.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Could not update SecObserve product member", err.Error())
		return
	}

	var refreshed model
	refreshed.fromAPI(updated)
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *productMemberResource) Delete(
	ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse,
) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteProductMember(ctx, state.ID.ValueInt64()); err != nil && !client.NotFound(err) {
		resp.Diagnostics.AddError("Could not delete SecObserve product member", err.Error())
	}
}

func (m *model) fromAPI(member client.ProductMember) {
	m.ID = types.Int64Value(member.ID)
	m.Product = types.Int64Value(member.Product)
	m.User = types.Int64Value(member.User)
	m.Role = types.StringValue(schemacommon.RoleName(member.Role))
}

// ImportState takes "<product id>/<user id>", the membership's natural key.
func (r *productMemberResource) ImportState(
	ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	productID, userID, found := strings.Cut(req.ID, "/")
	if !found {
		resp.Diagnostics.AddError(
			"Invalid import id",
			"Import a product member as \"<product id>/<user id>\", for example \"12/3\".",
		)
		return
	}

	parsedProduct, productErr := strconv.ParseInt(productID, 10, 64)
	parsedUser, userErr := strconv.ParseInt(userID, 10, 64)
	if productErr != nil || userErr != nil {
		resp.Diagnostics.AddError(
			"Invalid import id",
			"Both parts of \"<product id>/<user id>\" must be numeric.",
		)
		return
	}

	member, err := r.client.FindProductMember(ctx, parsedProduct, parsedUser)
	if err != nil {
		resp.Diagnostics.AddError("Could not import SecObserve product member", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), member.ID)...)
}
