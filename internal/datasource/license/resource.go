// Package license implements the secobserve_license data source.
package license

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tfutil"
)

var _ datasource.DataSourceWithConfigure = (*licenseDataSource)(nil)

// New returns the secobserve_license data source.
func New() datasource.DataSource {
	return &licenseDataSource{}
}

type licenseDataSource struct {
	client *client.Client
}

type model struct {
	ID            types.Int64  `tfsdk:"id"`
	SPDXID        types.String `tfsdk:"spdx_id"`
	Name          types.String `tfsdk:"name"`
	Reference     types.String `tfsdk:"reference"`
	IsOSIApproved types.Bool   `tfsdk:"is_osi_approved"`
	IsDeprecated  types.Bool   `tfsdk:"is_deprecated"`
}

func (d *licenseDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_license"
}

func (d *licenseDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = tfutil.DataSourceClient(req, resp)
}

func (d *licenseDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up a single SPDX license reference by its exact spdx_id, for example " +
			"`MIT` or `Apache-2.0`.\n\n" +
			"Licenses are system-managed: SecObserve seeds them from the SPDX license list and there is no " +
			"write endpoint. Use this data source to resolve an spdx_id to the numeric id required by " +
			"`secobserve_license_group` and `secobserve_license_policy_item`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Numeric id of the license.",
			},
			"spdx_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Exact SPDX license identifier, for example `MIT` or `Apache-2.0`.",
			},
			"name":      schema.StringAttribute{Computed: true, MarkdownDescription: "Human-readable name."},
			"reference": schema.StringAttribute{Computed: true, MarkdownDescription: "Reference URL or text for the license."},
			"is_osi_approved": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the license is OSI-approved. Null if unknown.",
			},
			"is_deprecated": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the SPDX identifier is deprecated. Null if unknown.",
			},
		},
	}
}

func (d *licenseDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config model
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := d.client.LicenseBySPDXID(ctx, config.SPDXID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Could not read SecObserve license", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model{
		ID:            types.Int64Value(found.ID),
		SPDXID:        types.StringValue(found.SPDXID),
		Name:          types.StringValue(found.Name),
		Reference:     types.StringValue(found.Reference),
		IsOSIApproved: tfutil.Bool(found.IsOSIApproved),
		IsDeprecated:  tfutil.Bool(found.IsDeprecated),
	})...)
}
