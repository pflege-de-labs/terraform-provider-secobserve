// Package branch implements the secobserve_branch data source.
package branch

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tfutil"
)

var _ datasource.DataSourceWithConfigure = (*branchDataSource)(nil)

// New returns the secobserve_branch data source.
func New() datasource.DataSource {
	return &branchDataSource{}
}

type branchDataSource struct {
	client *client.Client
}

type model struct {
	ID                   types.Int64  `tfsdk:"id"`
	Product              types.Int64  `tfsdk:"product"`
	Name                 types.String `tfsdk:"name"`
	NameWithProduct      types.String `tfsdk:"name_with_product"`
	IsDefaultBranch      types.Bool   `tfsdk:"is_default_branch"`
	HousekeepingProtect  types.Bool   `tfsdk:"housekeeping_protect"`
	LastImport           types.String `tfsdk:"last_import"`
	Purl                 types.String `tfsdk:"purl"`
	CPE23                types.String `tfsdk:"cpe23"`
	OSVLinuxDistribution types.String `tfsdk:"osv_linux_distribution"`
	OSVLinuxRelease      types.String `tfsdk:"osv_linux_release"`
}

func (d *branchDataSource) Metadata(
	_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_branch"
}

func (d *branchDataSource) Configure(
	_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse,
) {
	d.client = tfutil.DataSourceClient(req, resp)
}

func (d *branchDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up a branch by product and exact name. Branch names are unique per " +
			"product, so both are required.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{Computed: true, MarkdownDescription: "Numeric id of the branch."},
			"product": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "Id of the product the branch belongs to.",
			},
			"name": schema.StringAttribute{
				Required: true,
				MarkdownDescription: "Exact name of the branch. Matching is exact and case-sensitive, even " +
					"though SecObserve's own filter is a substring match.",
			},
			"name_with_product": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Branch name qualified with its product.",
			},
			"is_default_branch": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether this is the product's default branch.",
			},
			"housekeeping_protect": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the branch is exempt from housekeeping.",
			},
			"last_import": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "When observations were last imported for this branch.",
			},
			"purl":  schema.StringAttribute{Computed: true, MarkdownDescription: "Package URL identifying the branch."},
			"cpe23": schema.StringAttribute{Computed: true, MarkdownDescription: "CPE 2.3 name identifying the branch."},
			"osv_linux_distribution": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Linux distribution used when matching OSV advisories.",
			},
			"osv_linux_release": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Release of the Linux distribution.",
			},
		},
	}
}

func (d *branchDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config model
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := d.client.BranchByName(ctx, config.Product.ValueInt64(), config.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Could not read SecObserve branch", err.Error())
		return
	}

	state := model{
		ID:                   types.Int64Value(found.ID),
		Product:              types.Int64Value(found.Product),
		Name:                 types.StringValue(found.Name),
		NameWithProduct:      types.StringValue(found.NameWithProduct),
		IsDefaultBranch:      types.BoolValue(found.IsDefaultBranch),
		HousekeepingProtect:  types.BoolValue(found.HousekeepingProtect),
		LastImport:           tfutil.String(found.LastImport),
		Purl:                 types.StringValue(found.Purl),
		CPE23:                types.StringValue(found.CPE23),
		OSVLinuxDistribution: types.StringValue(found.OSVLinuxDistribution),
		OSVLinuxRelease:      types.StringValue(found.OSVLinuxRelease),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
