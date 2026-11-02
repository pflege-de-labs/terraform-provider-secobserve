package tfutil_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tfutil"
)

type item struct {
	id   int64
	name string
}

func (i item) GetID() int64    { return i.id }
func (i item) GetName() string { return i.name }

type itemModel struct {
	ID   types.Int64  `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func itemResourceSchema() rschema.Schema {
	return rschema.Schema{Attributes: map[string]rschema.Attribute{
		"id":   rschema.Int64Attribute{Computed: true},
		"name": rschema.StringAttribute{Computed: true},
	}}
}

func idIdentitySchema() resource.IdentitySchemaResponse {
	var resp resource.IdentitySchemaResponse
	tfutil.IDIdentitySchema(context.Background(), resource.IdentitySchemaRequest{}, &resp)
	return resp
}

func listRequest(limit int64, includeResource bool) list.ListRequest {
	return list.ListRequest{
		Limit:                  limit,
		IncludeResource:        includeResource,
		ResourceSchema:         itemResourceSchema(),
		ResourceIdentitySchema: idIdentitySchema().IdentitySchema,
	}
}

func toModel(_ context.Context, i item, _ *diag.Diagnostics) any {
	return itemModel{ID: types.Int64Value(i.id), Name: types.StringValue(i.name)}
}

func collect(stream *list.ListResultsStream) []list.ListResult {
	var results []list.ListResult
	for result := range stream.Results {
		results = append(results, result)
	}
	return results
}

func TestStreamList_HonoursLimit(t *testing.T) {
	ctx := context.Background()
	var stream list.ListResultsStream
	items := []item{{1, "one"}, {2, "two"}, {3, "three"}}

	tfutil.StreamList(ctx, listRequest(2, false), &stream, items, toModel)

	if got := len(collect(&stream)); got != 2 {
		t.Fatalf("got %d results, want 2", got)
	}
}

func TestStreamList_IdentityAndDisplayName(t *testing.T) {
	ctx := context.Background()
	var stream list.ListResultsStream

	tfutil.StreamList(ctx, listRequest(0, true), &stream, []item{{7, "seven"}}, toModel)

	results := collect(&stream)
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	result := results[0]
	if result.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", result.Diagnostics)
	}
	if result.DisplayName != "seven" {
		t.Errorf("DisplayName = %q, want %q", result.DisplayName, "seven")
	}
	var id types.Int64
	result.Identity.GetAttribute(ctx, path.Root("id"), &id)
	if id.ValueInt64() != 7 {
		t.Errorf("identity id = %v, want 7", id)
	}
	var name types.String
	result.Resource.GetAttribute(ctx, path.Root("name"), &name)
	if name.ValueString() != "seven" {
		t.Errorf("resource name = %v, want seven", name)
	}
}

func TestStreamList_SkipsResourceWhenNotIncluded(t *testing.T) {
	ctx := context.Background()
	var stream list.ListResultsStream
	called := false
	toResource := func(context.Context, item, *diag.Diagnostics) any {
		called = true
		return itemModel{}
	}

	tfutil.StreamList(ctx, listRequest(0, false), &stream, []item{{1, "one"}}, toResource)
	results := collect(&stream)

	if called {
		t.Error("toResource called although IncludeResource is false")
	}
	if len(results) != 1 || results[0].Diagnostics.HasError() {
		t.Fatalf("want one clean result, got %+v", results)
	}
}

func TestStreamList_ResourceDiagnosticsOnResult(t *testing.T) {
	ctx := context.Background()
	var stream list.ListResultsStream
	toResource := func(_ context.Context, _ item, diags *diag.Diagnostics) any {
		diags.AddError("boom", "lookup failed")
		return itemModel{}
	}

	tfutil.StreamList(ctx, listRequest(0, true), &stream, []item{{1, "one"}}, toResource)
	results := collect(&stream)

	if len(results) != 1 || !results[0].Diagnostics.HasError() {
		t.Fatalf("want one result carrying the error, got %+v", results)
	}
}
