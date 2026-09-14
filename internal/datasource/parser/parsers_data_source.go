package parser

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
)

// NewParsersDataSource returns the secobserve_parsers data source.
func NewParsersDataSource() datasource.DataSource {
	return &parsersDataSource{}
}

type parsersDataSource struct {
	client *client.Client
}

type parsersModel struct {
	Type    types.String  `tfsdk:"type"`
	Source  types.String  `tfsdk:"source"`
	Parsers []parserModel `tfsdk:"parsers"`
}

func (d *parsersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_parsers"
}

func (d *parsersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists the vulnerability scanner parsers available on the instance, optionally filtered by " +
			"type and source. Useful to discover the exact parser names to feed into `secobserve_parser`.",
		Attributes: map[string]schema.Attribute{
			"type": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "Only return parsers of this category: `SCA`, `SAST`, `DAST`, `IAST`, `Secrets`, " +
					"`Infrastructure`, `Other` or `Manual`.",
			},
			"source": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Only return parsers with this source: `API`, `File`, `Manual`, `Other` or `Unknown`.",
			},
			"parsers": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Matching parsers, ordered as returned by the API.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":          schema.Int64Attribute{Computed: true, MarkdownDescription: "Numeric id of the parser."},
						"name":        schema.StringAttribute{Computed: true, MarkdownDescription: "Name of the parser."},
						"type":        schema.StringAttribute{Computed: true, MarkdownDescription: "Scanner category."},
						"source":      schema.StringAttribute{Computed: true, MarkdownDescription: "How observations reach SecObserve."},
						"sbom":        schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether the parser consumes an SBOM."},
						"module_name": schema.StringAttribute{Computed: true, MarkdownDescription: "Python module implementing the parser."},
						"class_name":  schema.StringAttribute{Computed: true, MarkdownDescription: "Python class implementing the parser."},
					},
				},
			},
		},
	}
}

func (d *parsersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configuredClient(req, resp)
}

func (d *parsersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config parsersModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := d.client.Parsers(ctx, config.Type.ValueString(), config.Source.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Could not list SecObserve parsers", err.Error())
		return
	}

	config.Parsers = make([]parserModel, 0, len(found))
	for _, item := range found {
		config.Parsers = append(config.Parsers, parserModel{
			ID:         types.Int64Value(item.ID),
			Name:       types.StringValue(item.Name),
			Type:       types.StringValue(item.Type),
			Source:     types.StringValue(item.Source),
			SBOM:       types.BoolValue(item.SBOM),
			ModuleName: types.StringValue(item.ModuleName),
			ClassName:  types.StringValue(item.ClassName),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, config)...)
}

// configuredClient extracts the API client the provider handed down. Returns
// nil during the first Configure pass, when provider configuration is not yet
// resolved; the framework calls Configure again before Read.
func configuredClient(req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) *client.Client {
	if req.ProviderData == nil {
		return nil
	}
	apiClient, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected provider data type",
			fmt.Sprintf("Expected *client.Client, got %T. This is a bug in the provider.", req.ProviderData),
		)
		return nil
	}
	return apiClient
}
