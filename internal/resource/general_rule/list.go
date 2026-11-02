package general_rule

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	listschema "github.com/hashicorp/terraform-plugin-framework/list/schema"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tfutil"
)

var _ list.ListResourceWithConfigure = (*generalRuleResource)(nil)

// NewList returns the secobserve_general_rule list resource.
func NewList() list.ListResource {
	return &generalRuleResource{}
}

func (r *generalRuleResource) ListResourceConfigSchema(
	_ context.Context, _ list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse,
) {
	resp.Schema = listschema.Schema{
		MarkdownDescription: tfutil.ListDescription("Lists every SecObserve general rule. General rule names are not unique, so display names may repeat; the identity id is what import uses."),
	}
}

func (r *generalRuleResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	items, err := r.client.GeneralRules(ctx)
	if err != nil {
		stream.Results = tfutil.ListError("Could not list SecObserve general rules", err)
		return
	}

	tfutil.StreamList(ctx, req, stream, items, func(ctx context.Context, item client.Rule, diags *diag.Diagnostics) any {
		var m model
		m.fromAPI(ctx, item, diags)
		return m
	})
}
