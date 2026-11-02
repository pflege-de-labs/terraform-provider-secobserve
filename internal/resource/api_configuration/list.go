package api_configuration

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tfutil"
)

var _ list.ListResourceWithConfigure = (*apiConfigurationResource)(nil)

// NewList returns the secobserve_api_configuration list resource.
func NewList() list.ListResource {
	return &apiConfigurationResource{}
}

func (r *apiConfigurationResource) ListResourceConfigSchema(
	_ context.Context, _ list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse,
) {
	resp.Schema = tfutil.ProductFilterSchema("Lists the API configurations of one SecObserve product. `api_key` and `basic_auth_password` are sensitive, so generated configuration has them as null: fill them in before the first apply.")
}

func (r *apiConfigurationResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	var filter tfutil.ProductFilter
	diags := req.Config.Get(ctx, &filter)
	if diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	items, err := r.client.APIConfigurations(ctx, filter.Product.ValueInt64())
	if err != nil {
		stream.Results = tfutil.ListError("Could not list SecObserve API configurations", err)
		return
	}

	tfutil.StreamList(ctx, req, stream, items, func(_ context.Context, item client.APIConfiguration, _ *diag.Diagnostics) any {
		var m model
		m.fromAPI(item)
		return m
	})
}
