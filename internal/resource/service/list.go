package service

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tfutil"
)

var _ list.ListResourceWithConfigure = (*serviceResource)(nil)

// NewList returns the secobserve_service list resource.
func NewList() list.ListResource {
	return &serviceResource{}
}

func (r *serviceResource) ListResourceConfigSchema(
	_ context.Context, _ list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse,
) {
	resp.Schema = tfutil.ProductFilterSchema("Lists the services of one SecObserve product.")
}

func (r *serviceResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	var filter tfutil.ProductFilter
	diags := req.Config.Get(ctx, &filter)
	if diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	items, err := r.client.Services(ctx, filter.Product.ValueInt64())
	if err != nil {
		stream.Results = tfutil.ListError("Could not list SecObserve services", err)
		return
	}

	tfutil.StreamList(ctx, req, stream, items, func(_ context.Context, item client.Service, _ *diag.Diagnostics) any {
		var m model
		m.fromAPI(item)
		return m
	})
}
