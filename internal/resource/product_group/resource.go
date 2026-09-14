// Package product_group implements the secobserve_product_group resource.
package product_group

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tfutil"
)

var (
	_ resource.ResourceWithConfigure      = (*productGroupResource)(nil)
	_ resource.ResourceWithImportState    = (*productGroupResource)(nil)
	_ resource.ResourceWithValidateConfig = (*productGroupResource)(nil)
)

// New returns the secobserve_product_group resource.
func New() resource.Resource {
	return &productGroupResource{}
}

type productGroupResource struct {
	client *client.Client
}

func (r *productGroupResource) ValidateConfig(
	ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse,
) {
	var config model
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	config.SecurityGate.ValidateSecurityGate(&resp.Diagnostics)
	config.BranchPropagation.ValidateBranchPropagation(&resp.Diagnostics)
}

func (r *productGroupResource) Metadata(
	_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_product_group"
}

func (r *productGroupResource) Configure(
	_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse,
) {
	r.client = tfutil.ResourceClient(req, resp)
}

func (r *productGroupResource) Create(
	ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse,
) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	request := plan.toRequest(ctx, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateProductGroup(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError("Could not create SecObserve product group", err.Error())
		return
	}

	var state model
	state.fromAPI(ctx, created, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *productGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	group, err := r.client.ProductGroup(ctx, state.ID.ValueInt64())
	if err != nil {
		if client.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Could not read SecObserve product group", err.Error())
		return
	}

	var refreshed model
	refreshed.fromAPI(ctx, group, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *productGroupResource) Update(
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

	request := plan.toRequest(ctx, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateProductGroup(ctx, state.ID.ValueInt64(), request)
	if err != nil {
		resp.Diagnostics.AddError("Could not update SecObserve product group", err.Error())
		return
	}

	var refreshed model
	refreshed.fromAPI(ctx, updated, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *productGroupResource) Delete(
	ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse,
) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The name doubles as the delete confirmation parameter.
	err := r.client.DeleteProductGroup(ctx, state.ID.ValueInt64(), state.Name.ValueString())
	if err != nil {
		if client.NotFound(err) {
			return
		}
		if client.Conflict(err) {
			resp.Diagnostics.AddError(
				"SecObserve refused to delete the product group",
				"Something still references it, most likely a license policy that is protected against "+
					"deletion.\n\nUnderlying error: "+err.Error(),
			)
			return
		}
		resp.Diagnostics.AddError("Could not delete SecObserve product group", err.Error())
	}
}

// ImportState accepts either the numeric id or the product group's name.
// Names are unique across products and product groups, so either identifies
// exactly one object.
func (r *productGroupResource) ImportState(
	ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	if id, err := strconv.ParseInt(req.ID, 10, 64); err == nil {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
		return
	}

	group, err := r.client.ProductGroupByName(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Could not import SecObserve product group",
			"Pass either the numeric id or the exact name.\n\nUnderlying error: "+err.Error(),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), group.ID)...)
}
