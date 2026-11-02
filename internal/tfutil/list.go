package tfutil

import (
	"context"
	"iter"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	listschema "github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/client"
)

// StreamList emits one list result per item, up to req.Limit (0 means no
// limit). toResource converts an item into the resource's model and only runs
// when Terraform asked for full resource objects.
func StreamList[T client.Named](
	ctx context.Context,
	req list.ListRequest,
	stream *list.ListResultsStream,
	items []T,
	toResource func(context.Context, T, *diag.Diagnostics) any,
) {
	stream.Results = func(push func(list.ListResult) bool) {
		for i, item := range items {
			if req.Limit > 0 && int64(i) >= req.Limit {
				return
			}

			result := req.NewListResult(ctx)
			result.DisplayName = item.GetName()
			result.Diagnostics.Append(SetIdentityID(ctx, result.Identity, types.Int64Value(item.GetID()))...)

			if req.IncludeResource && !result.Diagnostics.HasError() {
				model := toResource(ctx, item, &result.Diagnostics)
				if !result.Diagnostics.HasError() {
					result.Diagnostics.Append(result.Resource.Set(ctx, model)...)
				}
			}

			if !push(result) {
				return
			}
		}
	}
}

// ListError is a result stream carrying a single error, for when the listing
// call itself failed.
func ListError(summary string, err error) iter.Seq[list.ListResult] {
	var diags diag.Diagnostics
	diags.AddError(summary, err.Error())
	return list.ListResultsStreamDiagnostics(diags)
}

// ProductFilter is the list config of resources listed per product.
type ProductFilter struct {
	Product types.Int64 `tfsdk:"product"`
}

// ProductFilterSchema is the list config schema matching ProductFilter.
func ProductFilterSchema(description string) listschema.Schema {
	return listschema.Schema{
		MarkdownDescription: description,
		Attributes: map[string]listschema.Attribute{
			"product": listschema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "Id of the product whose objects to list.",
			},
		},
	}
}
