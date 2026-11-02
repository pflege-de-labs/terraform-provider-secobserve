package license_policy

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	listschema "github.com/hashicorp/terraform-plugin-framework/list/schema"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tfutil"
)

var _ list.ListResourceWithConfigure = (*licensePolicyResource)(nil)

// NewList returns the secobserve_license_policy list resource.
func NewList() list.ListResource {
	return &licensePolicyResource{}
}

func (r *licensePolicyResource) ListResourceConfigSchema(
	_ context.Context, _ list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse,
) {
	resp.Schema = listschema.Schema{
		MarkdownDescription: "Lists every SecObserve license policy.",
	}
}

func (r *licensePolicyResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	items, err := r.client.LicensePolicies(ctx)
	if err != nil {
		stream.Results = tfutil.ListError("Could not list SecObserve license policies", err)
		return
	}

	tfutil.StreamList(ctx, req, stream, items, func(ctx context.Context, item client.LicensePolicy, diags *diag.Diagnostics) any {
		var m model
		m.fromAPI(ctx, item, diags)
		return m
	})
}
