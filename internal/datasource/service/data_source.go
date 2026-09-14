// Package service implements the secobserve_service data source.
package service

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jabbrwcky/terraform-provider-secobserve/internal/client"
	"github.com/jabbrwcky/terraform-provider-secobserve/internal/tfutil"
)

var _ datasource.DataSourceWithConfigure = (*serviceDataSource)(nil)

// New returns the secobserve_service data source.
func New() datasource.DataSource {
	return &serviceDataSource{}
}

type serviceDataSource struct {
	client *client.Client
}

type model struct {
	ID              types.Int64  `tfsdk:"id"`
	Product         types.Int64  `tfsdk:"product"`
	Name            types.String `tfsdk:"name"`
	NameWithProduct types.String `tfsdk:"name_with_product"`
}

func (d *serviceDataSource) Metadata(
	_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (d *serviceDataSource) Configure(
	_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse,
) {
	d.client = tfutil.DataSourceClient(req, resp)
}

func (d *serviceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up a service by product and exact name. Service names are unique per " +
			"product, so both are required.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{Computed: true, MarkdownDescription: "Numeric id of the service."},
			"product": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "Id of the product the service belongs to.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Exact name of the service.",
			},
			"name_with_product": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Service name qualified with its product.",
			},
		},
	}
}

func (d *serviceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config model
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := d.client.ServiceByName(ctx, config.Product.ValueInt64(), config.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Could not read SecObserve service", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model{
		ID:              types.Int64Value(found.ID),
		Product:         types.Int64Value(found.Product),
		Name:            types.StringValue(found.Name),
		NameWithProduct: types.StringValue(found.NameWithProduct),
	})...)
}
