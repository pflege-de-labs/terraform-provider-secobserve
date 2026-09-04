// Package parser contains the read-only data sources for SecObserve parsers.
package parser

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/jabbrwcky/terraform-provider-secobserve/internal/client"
)

// NewParserDataSource returns the secobserve_parser data source.
func NewParserDataSource() datasource.DataSource {
	return &parserDataSource{}
}

type parserDataSource struct {
	client *client.Client
}

type parserModel struct {
	ID         types.Int64  `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	Type       types.String `tfsdk:"type"`
	Source     types.String `tfsdk:"source"`
	SBOM       types.Bool   `tfsdk:"sbom"`
	ModuleName types.String `tfsdk:"module_name"`
	ClassName  types.String `tfsdk:"class_name"`
}

func (d *parserDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_parser"
}

func (d *parserDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up a single vulnerability scanner parser by name.\n\n" +
			"Parsers are system-managed: SecObserve registers them from its built-in parser modules at startup " +
			"and they cannot be created or modified through the API. Use this data source to resolve a parser " +
			"name to the numeric id required by `secobserve_api_configuration` and the rule resources.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Numeric id of the parser.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Exact name of the parser, for example `Trivy Filesystem` or `Dependency Track`.",
			},
			"type": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Scanner category: `SCA`, `SAST`, `DAST`, `IAST`, `Secrets`, `Infrastructure`, `Other` or `Manual`.",
			},
			"source": schema.StringAttribute{
				Computed: true,
				MarkdownDescription: "How observations reach SecObserve: `API`, `File`, `Manual`, `Other` or `Unknown`. " +
					"Only parsers with source `API` can be referenced by `secobserve_api_configuration`.",
			},
			"sbom": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the parser consumes an SBOM.",
			},
			"module_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Python module implementing the parser.",
			},
			"class_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Python class implementing the parser.",
			},
		},
	}
}

func (d *parserDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configuredClient(req, resp)
}

func (d *parserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config parserModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := d.client.Parser(ctx, config.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Could not read SecObserve parser", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, parserModel{
		ID:         types.Int64Value(found.ID),
		Name:       types.StringValue(found.Name),
		Type:       types.StringValue(found.Type),
		Source:     types.StringValue(found.Source),
		SBOM:       types.BoolValue(found.SBOM),
		ModuleName: types.StringValue(found.ModuleName),
		ClassName:  types.StringValue(found.ClassName),
	})...)
}
