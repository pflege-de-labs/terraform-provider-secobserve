// Package product_api_token implements the secobserve_product_api_token
// resource.
package product_api_token

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jabbrwcky/terraform-provider-secobserve/internal/client"
	"github.com/jabbrwcky/terraform-provider-secobserve/internal/schemacommon"
	"github.com/jabbrwcky/terraform-provider-secobserve/internal/tfutil"
)

var (
	_ resource.ResourceWithConfigure   = (*tokenResource)(nil)
	_ resource.ResourceWithImportState = (*tokenResource)(nil)
)

// New returns the secobserve_product_api_token resource.
func New() resource.Resource {
	return &tokenResource{}
}

type tokenResource struct {
	client *client.Client
}

type model struct {
	ID             types.Int64  `tfsdk:"id"`
	Product        types.Int64  `tfsdk:"product"`
	Role           types.String `tfsdk:"role"`
	Name           types.String `tfsdk:"name"`
	ExpirationDate types.String `tfsdk:"expiration_date"`
	Token          types.String `tfsdk:"token"`
}

func (r *tokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_product_api_token"
}

func (r *tokenResource) Configure(
	_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse,
) {
	r.client = tfutil.ResourceClient(req, resp)
}

func (r *tokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := map[string]schema.Attribute{
		"id": schema.Int64Attribute{
			Computed:            true,
			PlanModifiers:       []planmodifier.Int64{schemacommon.Int64UseStateForUnknown()},
			MarkdownDescription: "Numeric id of the token, used to revoke it.",
		},
		"product": schema.Int64Attribute{
			Required:      true,
			Validators:    []validator.Int64{int64validator.AtLeast(1)},
			PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			MarkdownDescription: "Id of the product the token is scoped to. Only one token per name and " +
				"product is allowed.",
		},
		"name": schema.StringAttribute{
			Required: true,
			// 32 characters, tighter than every other name in the API.
			Validators:          []validator.String{stringvalidator.LengthBetween(1, 32)},
			PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			MarkdownDescription: "Name of the token, at most **32** characters. Unique per product.",
		},
		"expiration_date": schema.StringAttribute{
			Optional:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			MarkdownDescription: "Date the token expires, as `YYYY-MM-DD`. Omit it for a token that never " +
				"expires. Must not be in the past.",
		},
		"token": schema.StringAttribute{
			Computed:      true,
			Sensitive:     true,
			PlanModifiers: []planmodifier.String{schemacommon.StringUseStateForUnknown()},
			MarkdownDescription: "The token value, for use as `Authorization: APIToken <token>`.\n\n" +
				"~> SecObserve returns this **once**, when the token is created, and offers no way to read " +
				"it back. It is kept in Terraform state; losing the state means recreating the token. " +
				"Imported tokens have this attribute unset for the same reason.",
		},
	}
	schemacommon.AddRole(attributes, "token")

	// Every attribute forces a replace: the endpoint offers only list, create
	// and delete, with no update at all.
	attributes["role"] = withRequiresReplace(attributes["role"])

	resp.Schema = schema.Schema{
		MarkdownDescription: "An API token scoped to a single product, for use by CI/CD pipelines uploading " +
			"scan results.\n\n" +
			"The endpoint supports only create, list and delete, so **any change replaces the token** and " +
			"issues a new secret. Consumers of the old value have to be updated.",
		Attributes: attributes,
	}
}

// withRequiresReplace adds the replace modifier to the shared role attribute,
// which has no modifiers of its own because the membership resources can patch
// the role in place.
func withRequiresReplace(attribute schema.Attribute) schema.Attribute {
	role, ok := attribute.(schema.StringAttribute)
	if !ok {
		return attribute
	}
	role.PlanModifiers = append(role.PlanModifiers, stringplanmodifier.RequiresReplace())
	return role
}

func (r *tokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	secret, err := r.client.CreateProductAPIToken(ctx, client.ProductAPITokenRequest{
		Product:        plan.Product.ValueInt64(),
		Role:           schemacommon.RoleValue(plan.Role.ValueString()),
		Name:           plan.Name.ValueString(),
		ExpirationDate: tfutil.StringPtr(plan.ExpirationDate),
	})
	if err != nil {
		resp.Diagnostics.AddError("Could not create SecObserve product API token", err.Error())
		return
	}

	// The create response carries only the secret, so the id has to be looked
	// up. Doing it now rather than on the next read keeps the token usable
	// even if the following plan never runs.
	created, err := r.client.FindProductAPIToken(ctx, plan.Product.ValueInt64(), plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Created the SecObserve product API token but could not read it back",
			"The token exists and its secret is below, but Terraform could not determine its id, so it is "+
				"not tracked in state. Revoke it in SecObserve before applying again.\n\nUnderlying error: "+
				err.Error(),
		)
		return
	}

	state := plan
	state.fromAPI(created)
	state.Token = types.StringValue(secret)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Read lists the product's tokens and matches on the name: the endpoint has no
// retrieve action.
func (r *tokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	token, err := r.client.FindProductAPIToken(ctx, state.Product.ValueInt64(), state.Name.ValueString())
	if err != nil {
		// The token is gone either if the product 404s or if no token of that
		// name is left in the product's list.
		var notFound *client.ErrNotFound
		if client.NotFound(err) || errors.As(err, &notFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Could not read SecObserve product API token", err.Error())
		return
	}

	refreshed := state
	refreshed.fromAPI(token)
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

// Update is unreachable: every attribute forces a replace. It exists only to
// satisfy the interface.
func (r *tokenResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"SecObserve product API tokens cannot be updated",
		"The API offers no update for product API tokens, so every change should have forced a replace. "+
			"This is a bug in the provider.",
	)
}

func (r *tokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteProductAPIToken(ctx, state.ID.ValueInt64()); err != nil && !client.NotFound(err) {
		resp.Diagnostics.AddError("Could not delete SecObserve product API token", err.Error())
	}
}

func (m *model) fromAPI(token client.ProductAPIToken) {
	m.ID = types.Int64Value(token.ID)
	m.Product = types.Int64Value(token.Product)
	m.Name = types.StringValue(token.Name)
	m.Role = types.StringValue(schemacommon.RoleName(token.Role))
	m.ExpirationDate = tfutil.String(token.ExpirationDate)
}

// ImportState takes "<product id>/<token name>". The secret stays unset: it is
// unrecoverable.
func (r *tokenResource) ImportState(
	ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	productID, name, found := strings.Cut(req.ID, "/")
	if !found {
		resp.Diagnostics.AddError(
			"Invalid import id",
			"Import a product API token as \"<product id>/<token name>\", for example \"12/ci\".",
		)
		return
	}

	parsedProduct, err := strconv.ParseInt(productID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid import id",
			"The product part of \"<product id>/<token name>\" must be numeric: "+err.Error(),
		)
		return
	}

	token, err := r.client.FindProductAPIToken(ctx, parsedProduct, name)
	if err != nil {
		resp.Diagnostics.AddError("Could not import SecObserve product API token", err.Error())
		return
	}

	var state model
	state.fromAPI(token)
	// Unrecoverable by construction: SecObserve returns the secret only once.
	state.Token = types.StringNull()
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	resp.Diagnostics.AddWarning(
		"Imported product API token without its secret",
		"SecObserve returns a token's secret only when it is created, so the token attribute is null. "+
			"Anything that needs the value has to use the token stored outside Terraform, or the resource "+
			"has to be recreated to issue a new one.",
	)
}
