// Package product implements the secobserve_product resource.
package product

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/jabbrwcky/terraform-provider-secobserve/internal/client"
	"github.com/jabbrwcky/terraform-provider-secobserve/internal/tfutil"
)

var (
	_ resource.ResourceWithConfigure      = (*productResource)(nil)
	_ resource.ResourceWithImportState    = (*productResource)(nil)
	_ resource.ResourceWithValidateConfig = (*productResource)(nil)
)

// New returns the secobserve_product resource.
func New() resource.Resource {
	return &productResource{}
}

type productResource struct {
	client *client.Client
}

func (r *productResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_product"
}

func (r *productResource) Configure(
	_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse,
) {
	r.client = tfutil.ResourceClient(req, resp)
}

func (r *productResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := plan.toRequest(ctx, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateProduct(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError("Could not create SecObserve product", err.Error())
		return
	}

	var state model
	state.fromAPI(ctx, created, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *productResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	product, err := r.client.Product(ctx, state.ID.ValueInt64())
	if err != nil {
		if client.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Could not read SecObserve product", err.Error())
		return
	}

	var refreshed model
	refreshed.fromAPI(ctx, product, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *productResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
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

	updated, err := r.client.UpdateProduct(ctx, state.ID.ValueInt64(), request)
	if err != nil {
		resp.Diagnostics.AddError("Could not update SecObserve product", err.Error())
		return
	}

	var refreshed model
	refreshed.fromAPI(ctx, updated, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *productResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The name doubles as the delete confirmation parameter.
	err := r.client.DeleteProduct(ctx, state.ID.ValueInt64(), state.Name.ValueString())
	if err != nil {
		if client.NotFound(err) {
			return
		}
		if client.Conflict(err) {
			resp.Diagnostics.AddError(
				"SecObserve refused to delete the product",
				"Something still references it, most likely a license policy or a service that observations "+
					"point at.\n\nUnderlying error: "+err.Error(),
			)
			return
		}
		resp.Diagnostics.AddError("Could not delete SecObserve product", err.Error())
	}
}

// ImportState accepts either the numeric id or the product's name.
func (r *productResource) ImportState(
	ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	if id, err := strconv.ParseInt(req.ID, 10, 64); err == nil {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
		return
	}

	product, err := r.client.ProductByName(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Could not import SecObserve product",
			"Pass either the numeric id or the exact name.\n\nUnderlying error: "+err.Error(),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), product.ID)...)
}
