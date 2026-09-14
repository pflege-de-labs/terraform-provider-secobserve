// Package product implements the secobserve_product data source.
package product

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
	_ datasource.DataSourceWithConfigure        = (*productDataSource)(nil)
	_ datasource.DataSourceWithConfigValidators = (*productDataSource)(nil)
)

// New returns the secobserve_product data source.
func New() datasource.DataSource {
	return &productDataSource{}
}

type productDataSource struct {
	client *client.Client
}

type model struct {
	ID               types.Int64  `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	ProductGroup     types.Int64  `tfsdk:"product_group"`
	ProductGroupName types.String `tfsdk:"product_group_name"`

	RepositoryPrefix            types.String `tfsdk:"repository_prefix"`
	RepositoryDefaultBranch     types.Int64  `tfsdk:"repository_default_branch"`
	RepositoryDefaultBranchName types.String `tfsdk:"repository_default_branch_name"`

	Purl  types.String `tfsdk:"purl"`
	CPE23 types.String `tfsdk:"cpe23"`

	LicensePolicy      types.Int64 `tfsdk:"license_policy"`
	SecurityGateActive types.Bool  `tfsdk:"security_gate_active"`
	SecurityGatePassed types.Bool  `tfsdk:"security_gate_passed"`
	IssueTrackerActive types.Bool  `tfsdk:"issue_tracker_active"`

	dscommon.ObservationCounts
	dscommon.ContentFlags
}

func (d *productDataSource) Metadata(
	_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_product"
}

func (d *productDataSource) Configure(
	_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse,
) {
	d.client = tfutil.DataSourceClient(req, resp)
}

// ConfigValidators enforces exactly one lookup key.
func (d *productDataSource) ConfigValidators(_ context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.ExactlyOneOf(path.MatchRoot("id"), path.MatchRoot("name")),
	}
}

func (d *productDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := map[string]schema.Attribute{
		"id": schema.Int64Attribute{
			Optional:            true,
			Computed:            true,
			MarkdownDescription: "Numeric id of the product. Supply either this or `name`.",
		},
		"name": schema.StringAttribute{
			Optional: true,
			Computed: true,
			MarkdownDescription: "Exact name of the product. Supply either this or `id`.\n\n" +
				"Matching is exact and case-sensitive, even though SecObserve's own filter is a substring " +
				"match.",
		},
		"description":        schema.StringAttribute{Computed: true, MarkdownDescription: "Free-text description."},
		"product_group":      schema.Int64Attribute{Computed: true, MarkdownDescription: "Id of the product group, if any."},
		"product_group_name": schema.StringAttribute{Computed: true, MarkdownDescription: "Name of the product group, if any."},
		"repository_prefix":  schema.StringAttribute{Computed: true, MarkdownDescription: "Base URL of the source repository."},
		"repository_default_branch": schema.Int64Attribute{
			Computed:            true,
			MarkdownDescription: "Id of the default branch.",
		},
		"repository_default_branch_name": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Name of the default branch.",
		},
		"purl":           schema.StringAttribute{Computed: true, MarkdownDescription: "Package URL identifying the product."},
		"cpe23":          schema.StringAttribute{Computed: true, MarkdownDescription: "CPE 2.3 name identifying the product."},
		"license_policy": schema.Int64Attribute{Computed: true, MarkdownDescription: "Id of the applied license policy."},
		"security_gate_active": schema.BoolAttribute{
			Computed:            true,
			MarkdownDescription: "Whether the security gate is evaluated. Null means the setting is inherited.",
		},
		"security_gate_passed": schema.BoolAttribute{
			Computed:            true,
			MarkdownDescription: "Whether the product currently passes its security gate.",
		},
		"issue_tracker_active": schema.BoolAttribute{
			Computed:            true,
			MarkdownDescription: "Whether observations are pushed to an issue tracker.",
		},
	}
	dscommon.AddObservationCounts(attributes)
	dscommon.AddContentFlags(attributes)

	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up a product by id or exact name.\n\n" +
			"Use it to reference a product managed elsewhere, and to read the runtime information that the " +
			"`secobserve_product` resource deliberately does not carry: the active observation counts and " +
			"the flags describing what kinds of findings the product has.",
		Attributes: attributes,
	}
}

func (d *productDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
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
		found, err = d.client.Product(ctx, config.ID.ValueInt64())
	} else {
		found, err = d.client.ProductByName(ctx, config.Name.ValueString())
		if err == nil {
			// The name lookup goes through the list endpoint, whose serializer
			// omits several attributes; re-read the detail representation.
			found, err = d.client.Product(ctx, found.ID)
		}
	}
	if err != nil {
		resp.Diagnostics.AddError("Could not read SecObserve product", err.Error())
		return
	}

	var state model
	state.fromAPI(found)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (m *model) fromAPI(product client.Product) {
	m.ID = types.Int64Value(product.ID)
	m.Name = types.StringValue(product.Name)
	m.Description = types.StringValue(product.Description)
	m.ProductGroup = tfutil.Int64(product.ProductGroup)
	m.ProductGroupName = tfutil.String(product.ProductGroupName)

	m.RepositoryPrefix = types.StringValue(product.RepositoryPrefix)
	m.RepositoryDefaultBranch = tfutil.Int64(product.RepositoryDefaultBranch)
	m.RepositoryDefaultBranchName = tfutil.String(product.RepositoryDefaultBranchName)

	m.Purl = types.StringValue(product.Purl)
	m.CPE23 = types.StringValue(product.CPE23)

	m.LicensePolicy = tfutil.Int64(product.LicensePolicy)
	m.SecurityGateActive = tfutil.Bool(product.SecurityGateActive)
	m.SecurityGatePassed = tfutil.Bool(product.SecurityGatePassed)
	m.IssueTrackerActive = types.BoolValue(product.IssueTrackerActive)

	m.ObservationCounts.FromAPI(product)
	m.ContentFlags.FromAPI(product)
}
