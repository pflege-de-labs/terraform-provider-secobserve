package user

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	listschema "github.com/hashicorp/terraform-plugin-framework/list/schema"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tfutil"
)

var _ list.ListResourceWithConfigure = (*userResource)(nil)

// NewList returns the secobserve_user list resource.
func NewList() list.ListResource {
	return &userResource{}
}

func (r *userResource) ListResourceConfigSchema(
	_ context.Context, _ list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse,
) {
	resp.Schema = listschema.Schema{
		MarkdownDescription: tfutil.ListDescription("Lists every SecObserve user."),
	}
}

func (r *userResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	items, err := r.client.Users(ctx)
	if err != nil {
		stream.Results = tfutil.ListError("Could not list SecObserve users", err)
		return
	}

	tfutil.StreamList(ctx, req, stream, items, func(_ context.Context, item client.User, _ *diag.Diagnostics) any {
		var m model
		m.fromAPI(item)
		return m
	})
}
