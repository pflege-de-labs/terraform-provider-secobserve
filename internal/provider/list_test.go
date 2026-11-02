package provider_test

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/list"
	fwprovider "github.com/hashicorp/terraform-plugin-framework/provider"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/provider"
)

// Terraform stops at 100 results unless a list block sets limit; bulk
// imports of larger instances silently come up short without this note.
func TestListResourcesDocumentLimit(t *testing.T) {
	ctx := context.Background()
	p := provider.New("test")().(fwprovider.ProviderWithListResources)
	for _, newList := range p.ListResources(ctx) {
		var resp list.ListResourceSchemaResponse
		newList().ListResourceConfigSchema(ctx, list.ListResourceSchemaRequest{}, &resp)
		if !strings.Contains(resp.Schema.MarkdownDescription, "`limit`") {
			t.Errorf("description lacks the limit note: %q", resp.Schema.MarkdownDescription)
		}
	}
}
