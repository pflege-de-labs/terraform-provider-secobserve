// Package service implements the secobserve_service resource.
package service

import (
	"context"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jabbrwcky/terraform-provider-secobserve/internal/client"
	"github.com/jabbrwcky/terraform-provider-secobserve/internal/schemacommon"
	"github.com/jabbrwcky/terraform-provider-secobserve/internal/tfutil"
)

var (
	_ resource.ResourceWithConfigure   = (*serviceResource)(nil)
	_ resource.ResourceWithImportState = (*serviceResource)(nil)
)

// New returns the secobserve_service resource.
func New() resource.Resource {
	return &serviceResource{}
}

type serviceResource struct {
	client *client.Client
}

type model struct {
	ID              types.Int64  `tfsdk:"id"`
	Product         types.Int64  `tfsdk:"product"`
	Name            types.String `tfsdk:"name"`
	NameWithProduct types.String `tfsdk:"name_with_product"`
}

func (r *serviceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (r *serviceResource) Configure(
	_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse,
) {
	r.client = tfutil.ResourceClient(req, resp)
}

func (r *serviceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A service is a named grouping of observations inside a product, used when one " +
			"product covers several deployable services.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{schemacommon.Int64UseStateForUnknown()},
				MarkdownDescription: "Numeric id of the service.",
			},
			"product": schema.Int64Attribute{
				Required:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
				MarkdownDescription: "Id of the product this service belongs to. Immutable: a change replaces " +
					"the service.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthBetween(1, 255)},
				MarkdownDescription: "Name of the service. Unique per product, not globally.",
			},
			"name_with_product": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Service name qualified with its product, as SecObserve displays it.",
			},
		},
	}
}

func (m *model) fromAPI(service client.Service) {
	m.ID = types.Int64Value(service.ID)
	m.Product = types.Int64Value(service.Product)
	m.Name = types.StringValue(service.Name)
	m.NameWithProduct = types.StringValue(service.NameWithProduct)
}

func (r *serviceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateService(ctx, client.ServiceRequest{
		Product: plan.Product.ValueInt64(),
		Name:    plan.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Could not create SecObserve service", err.Error())
		return
	}

	var state model
	state.fromAPI(created)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *serviceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	service, err := r.client.Service(ctx, state.ID.ValueInt64())
	if err != nil {
		if client.NotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Could not read SecObserve service", err.Error())
		return
	}

	var refreshed model
	refreshed.fromAPI(service)
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *serviceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
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

	updated, err := r.client.UpdateService(ctx, state.ID.ValueInt64(), client.ServiceRequest{
		Product: plan.Product.ValueInt64(),
		Name:    plan.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Could not update SecObserve service", err.Error())
		return
	}

	var refreshed model
	refreshed.fromAPI(updated)
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *serviceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteService(ctx, state.ID.ValueInt64())
	if err != nil {
		if client.NotFound(err) {
			return
		}
		if client.Conflict(err) {
			resp.Diagnostics.AddError(
				"SecObserve refused to delete the service",
				"Observations still reference it as their origin service. Delete or reassign those "+
					"observations first.\n\nUnderlying error: "+err.Error(),
			)
			return
		}
		resp.Diagnostics.AddError("Could not delete SecObserve service", err.Error())
	}
}

// ImportState takes "<product id>/<service name>", because service names are
// only unique within a product.
func (r *serviceResource) ImportState(
	ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	productID, name, found := strings.Cut(req.ID, "/")
	if !found {
		resp.Diagnostics.AddError(
			"Invalid import id",
			"Import a service as \"<product id>/<service name>\", for example \"12/api\".",
		)
		return
	}

	parsedProduct, err := strconv.ParseInt(productID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid import id",
			"The product part of \"<product id>/<service name>\" must be numeric: "+err.Error(),
		)
		return
	}

	service, err := r.client.ServiceByName(ctx, parsedProduct, name)
	if err != nil {
		resp.Diagnostics.AddError("Could not import SecObserve service", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), service.ID)...)
}
