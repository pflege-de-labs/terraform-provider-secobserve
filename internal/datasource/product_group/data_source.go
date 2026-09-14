// Package product_group implements the secobserve_product_group data source.
package product_group

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
	dscommon "github.com/pflege-de-labs/terraform-provider-secobserve/internal/datasource/common"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tfutil"
)

var (
	_ datasource.DataSourceWithConfigure        = (*productGroupDataSource)(nil)
	_ datasource.DataSourceWithConfigValidators = (*productGroupDataSource)(nil)
)

// New returns the secobserve_product_group data source.
func New() datasource.DataSource {
	return &productGroupDataSource{}
}

type productGroupDataSource struct {
	client *client.Client
}

type model struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`

	LicensePolicy      types.Int64 `tfsdk:"license_policy"`
	SecurityGateActive types.Bool  `tfsdk:"security_gate_active"`
	ProductsCount      types.Int64 `tfsdk:"products_count"`

	dscommon.ObservationCounts
}

func (d *productGroupDataSource) Metadata(
	_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_product_group"
}

func (d *productGroupDataSource) Configure(
	_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse,
) {
	d.client = tfutil.DataSourceClient(req, resp)
}

func (d *productGroupDataSource) ConfigValidators(_ context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(path.MatchRoot("id"), path.MatchRoot("name")),
	}
}

func (d *productGroupDataSource) Schema(
	_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse,
) {
	attributes := map[string]schema.Attribute{
		"id": schema.Int64Attribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Numeric id of the product group. Supply either this or `name`.",
		},
		"name": schema.StringAttribute{
			Optional: true,
			Computed: true,
			MarkdownDescription: "Exact name of the product group. Supply either this or `id`. Matching is " +
				"exact and case-sensitive.",
		},
		"description":    schema.StringAttribute{Computed: true, MarkdownDescription: "Free-text description."},
		"license_policy": schema.Int64Attribute{Computed: true, MarkdownDescription: "Id of the applied license policy."},
		"security_gate_active": schema.BoolAttribute{
			Computed:            true,
			MarkdownDescription: "Whether the security gate is evaluated. Null means the setting is inherited.",
		},
		"products_count": schema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "Number of products in the group.",
		},
	}
	dscommon.AddObservationCounts(attributes)

	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up a product group by id or exact name.\n\n" +
			"Note that products and product groups share one name space, so a name identifies at most one " +
			"of either.",
		Attributes: attributes,
	}
}

func (d *productGroupDataSource) Read(
	ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse,
) {
	var config model
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var (
		found client.Product
		err   error
	)
	if !config.ID.IsNull() {
		found, err = d.client.ProductGroup(ctx, config.ID.ValueInt64())
	} else {
		found, err = d.client.ProductGroupByName(ctx, config.Name.ValueString())
		if err == nil {
			// The name lookup goes through the list endpoint, whose serializer
			// omits several attributes; re-read the detail representation.
			found, err = d.client.ProductGroup(ctx, found.ID)
		}
	}
	if err != nil {
		resp.Diagnostics.AddError("Could not read SecObserve product group", err.Error())
		return
	}

	var state model
	state.ID = types.Int64Value(found.ID)
	state.Name = types.StringValue(found.Name)
	state.Description = types.StringValue(found.Description)
	state.LicensePolicy = tfutil.Int64(found.LicensePolicy)
	state.SecurityGateActive = tfutil.Bool(found.SecurityGateActive)
	state.ProductsCount = tfutil.Int64(found.ProductsCount)
	state.ObservationCounts.FromAPI(found)

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
