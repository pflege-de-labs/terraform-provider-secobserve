// Package vex_counter implements the secobserve_vex_counter data source.
package vex_counter //nolint:revive // the package name mirrors the Terraform data source name

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tfutil"
)

var _ datasource.DataSourceWithConfigure = (*vexCounterDataSource)(nil)

// New returns the secobserve_vex_counter data source.
func New() datasource.DataSource {
	return &vexCounterDataSource{}
}

type vexCounterDataSource struct {
	client *client.Client
}

type model struct {
	ID               types.Int64  `tfsdk:"id"`
	DocumentIDPrefix types.String `tfsdk:"document_id_prefix"`
	Year             types.Int64  `tfsdk:"year"`
	Counter          types.Int64  `tfsdk:"counter"`
}

func (d *vexCounterDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vex_counter"
}

func (d *vexCounterDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = tfutil.DataSourceClient(req, resp)
}

func (d *vexCounterDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads the current VEX document id sequence counter for a " +
			"(document_id_prefix, year) pair.\n\n" +
			"~> **This is read-only by design, not an oversight.** `counter` is incremented server-side on " +
			"every CSAF/OpenVEX document export, so a Terraform-managed resource would show drift on every " +
			"export and a write could rewind the sequence, producing duplicate document ids. There is no " +
			"`secobserve_vex_counter` resource. Rows are created implicitly the first time a document with a " +
			"new prefix is exported, so a lookup for a combination that has never been exported returns not found.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Numeric id of the counter row.",
			},
			"document_id_prefix": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Exact document id prefix used when exporting VEX documents.",
			},
			"year": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "Year component of the counter's key.",
			},
			"counter": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Current sequence value: the next exported document for this prefix and year gets counter + 1.",
			},
		},
	}
}

func (d *vexCounterDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config model
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := d.client.VEXCounterByPrefixAndYear(ctx, config.DocumentIDPrefix.ValueString(), config.Year.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Could not read SecObserve VEX counter", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model{
		ID:               types.Int64Value(found.ID),
		DocumentIDPrefix: types.StringValue(found.DocumentIDPrefix),
		Year:             types.Int64Value(found.Year),
		Counter:          types.Int64Value(found.Counter),
	})...)
}
