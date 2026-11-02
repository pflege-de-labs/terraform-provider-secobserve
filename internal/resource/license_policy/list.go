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
		MarkdownDescription: tfutil.ListDescription("Lists every SecObserve license policy."),
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
		// The list serializer omits fields the detail one has; generated config would clear them.
		detail, err := r.client.LicensePolicy(ctx, item.ID)
		if err != nil {
			diags.AddError("Could not read SecObserve license policy", err.Error())
			return m
		}
		m.fromAPI(ctx, detail, diags)
		return m
	})
}
