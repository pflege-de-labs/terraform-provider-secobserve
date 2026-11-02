package product_group

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	listschema "github.com/hashicorp/terraform-plugin-framework/list/schema"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tfutil"
)

var _ list.ListResourceWithConfigure = (*productGroupResource)(nil)

// NewList returns the secobserve_product_group list resource.
func NewList() list.ListResource {
	return &productGroupResource{}
}

func (r *productGroupResource) ListResourceConfigSchema(
	_ context.Context, _ list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse,
) {
	resp.Schema = listschema.Schema{
		MarkdownDescription: tfutil.ListDescription("Lists every SecObserve product group. Generated configuration never contains `security_gate` or `repository_branch_housekeeping`: add them before the first apply, or an update resets them to inherit."),
	}
}

func (r *productGroupResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	items, err := r.client.ProductGroups(ctx)
	if err != nil {
		stream.Results = tfutil.ListError("Could not list SecObserve product groups", err)
		return
	}

	tfutil.StreamList(ctx, req, stream, items, func(ctx context.Context, item client.Product, diags *diag.Diagnostics) any {
		var m model
		// The list serializer omits fields the detail one has; generated config would clear them.
		detail, err := r.client.ProductGroup(ctx, item.ID)
		if err != nil {
			diags.AddError("Could not read SecObserve product group", err.Error())
			return m
		}
		m.fromAPI(ctx, detail, diags)
		return m
	})
}
