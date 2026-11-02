package branch

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tfutil"
)

var _ list.ListResourceWithConfigure = (*branchResource)(nil)

// NewList returns the secobserve_branch list resource.
func NewList() list.ListResource {
	return &branchResource{}
}

func (r *branchResource) ListResourceConfigSchema(
	_ context.Context, _ list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse,
) {
	resp.Schema = tfutil.ProductFilterSchema("Lists the branches of one SecObserve product.")
}

func (r *branchResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	var filter tfutil.ProductFilter
	diags := req.Config.Get(ctx, &filter)
	if diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	items, err := r.client.Branches(ctx, filter.Product.ValueInt64())
	if err != nil {
		stream.Results = tfutil.ListError("Could not list SecObserve branches", err)
		return
	}

	tfutil.StreamList(ctx, req, stream, items, func(_ context.Context, item client.Branch, _ *diag.Diagnostics) any {
		var m model
		m.fromAPI(item)
		return m
	})
}
