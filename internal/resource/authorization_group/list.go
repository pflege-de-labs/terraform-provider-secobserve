package authorization_group

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	listschema "github.com/hashicorp/terraform-plugin-framework/list/schema"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tfutil"
)

var _ list.ListResourceWithConfigure = (*authorizationGroupResource)(nil)

// NewList returns the secobserve_authorization_group list resource.
func NewList() list.ListResource {
	return &authorizationGroupResource{}
}

func (r *authorizationGroupResource) ListResourceConfigSchema(
	_ context.Context, _ list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse,
) {
	resp.Schema = listschema.Schema{
		MarkdownDescription: tfutil.ListDescription("Lists every SecObserve authorization group."),
	}
}

func (r *authorizationGroupResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	items, err := r.client.AuthorizationGroups(ctx)
	if err != nil {
		stream.Results = tfutil.ListError("Could not list SecObserve authorization groups", err)
		return
	}

	tfutil.StreamList(ctx, req, stream, items, func(_ context.Context, item client.AuthorizationGroup, _ *diag.Diagnostics) any {
		var m model
		m.fromAPI(item)
		return m
	})
}
