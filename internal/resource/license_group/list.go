package license_group //nolint:revive // the package name mirrors the Terraform resource name

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	listschema "github.com/hashicorp/terraform-plugin-framework/list/schema"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tfutil"
)

var _ list.ListResourceWithConfigure = (*licenseGroupResource)(nil)

// NewList returns the secobserve_license_group list resource.
func NewList() list.ListResource {
	return &licenseGroupResource{}
}

func (r *licenseGroupResource) ListResourceConfigSchema(
	_ context.Context, _ list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse,
) {
	resp.Schema = listschema.Schema{
		MarkdownDescription: tfutil.ListDescription("Lists every SecObserve license group, including the ScanCode LicenseDB groups " +
			"SecObserve maintains itself. With `include_resource = true` each group costs one extra request " +
			"for its licenses."),
	}
}

func (r *licenseGroupResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	groups, err := r.client.LicenseGroups(ctx)
	if err != nil {
		stream.Results = tfutil.ListError("Could not list SecObserve license groups", err)
		return
	}

	tfutil.StreamList(ctx, req, stream, groups, func(ctx context.Context, group client.LicenseGroup, diags *diag.Diagnostics) any {
		var m model
		licenses, err := r.client.ListLicensesInGroup(ctx, group.ID)
		if err != nil {
			diags.AddError("Could not list licenses in SecObserve license group", err.Error())
			return m
		}
		m.fromAPI(ctx, group, licenses, diags)
		return m
	})
}
