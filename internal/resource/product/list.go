package product

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	listschema "github.com/hashicorp/terraform-plugin-framework/list/schema"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tfutil"
)

var _ list.ListResourceWithConfigure = (*productResource)(nil)

// NewList returns the secobserve_product list resource.
func NewList() list.ListResource {
	return &productResource{}
}

func (r *productResource) ListResourceConfigSchema(
	_ context.Context, _ list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse,
) {
	resp.Schema = listschema.Schema{
		MarkdownDescription: "Lists every SecObserve product (product groups excluded). " +
			"Generated configuration never contains `security_gate` or `repository_branch_housekeeping`: " +
			"add them before the first apply, or an update resets them to inherit. " +
			"Sensitive values such as `issue_tracker_api_key` are generated as null and must be filled in.",
	}
}

func (r *productResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	products, err := r.client.Products(ctx)
	if err != nil {
		stream.Results = tfutil.ListError("Could not list SecObserve products", err)
		return
	}

	tfutil.StreamList(ctx, req, stream, products, func(ctx context.Context, product client.Product, diags *diag.Diagnostics) any {
		var m model
		m.fromAPI(ctx, product, diags)
		return m
	})
}
