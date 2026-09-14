// Package tfutil holds plumbing shared by every resource and data source:
// client hand-off, and conversion between Terraform and JSON representations.
package tfutil

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
)

// ResourceClient extracts the API client the provider handed down.
//
// It returns nil during the framework's first Configure pass, when provider
// configuration is not yet resolved. Configure is called again before any CRUD
// method, so callers only need to guard against a nil client in tests.
func ResourceClient(req resource.ConfigureRequest, resp *resource.ConfigureResponse) *client.Client {
	return extract(req.ProviderData, &resp.Diagnostics)
}

// DataSourceClient is ResourceClient for data sources.
func DataSourceClient(req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) *client.Client {
	return extract(req.ProviderData, &resp.Diagnostics)
}

func extract(providerData any, diags *diag.Diagnostics) *client.Client {
	if providerData == nil {
		return nil
	}
	apiClient, ok := providerData.(*client.Client)
	if !ok {
		diags.AddError(
			"Unexpected provider data type",
			fmt.Sprintf("Expected *client.Client, got %T. This is a bug in the provider.", providerData),
		)
		return nil
	}
	return apiClient
}
