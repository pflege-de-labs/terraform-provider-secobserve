package product_rule

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tfutil"
)

var _ list.ListResourceWithConfigure = (*productRuleResource)(nil)

// NewList returns the secobserve_product_rule list resource.
func NewList() list.ListResource {
	return &productRuleResource{}
}

func (r *productRuleResource) ListResourceConfigSchema(
	_ context.Context, _ list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse,
) {
	resp.Schema = tfutil.ProductFilterSchema("Lists the rules of one SecObserve product.")
}

func (r *productRuleResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	var filter tfutil.ProductFilter
	diags := req.Config.Get(ctx, &filter)
	if diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	items, err := r.client.ProductRules(ctx, filter.Product.ValueInt64())
	if err != nil {
		stream.Results = tfutil.ListError("Could not list SecObserve product rules", err)
		return
	}

	tfutil.StreamList(ctx, req, stream, items, func(ctx context.Context, item client.Rule, diags *diag.Diagnostics) any {
		var m model
		m.fromAPI(ctx, item, diags)
		return m
	})
}
