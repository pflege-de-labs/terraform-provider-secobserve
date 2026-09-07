// Package tftest holds helpers shared by the provider's tests.
package tftest

import (
	"context"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// ResourceSchema instantiates a resource's schema and fails the test if the
// framework considers it invalid.
func ResourceSchema(t *testing.T, res fwresource.Resource) schema.Schema {
	t.Helper()

	ctx := context.Background()
	resp := &fwresource.SchemaResponse{}
	res.Schema(ctx, fwresource.SchemaRequest{}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("schema has errors: %v", resp.Diagnostics.Errors())
	}
	if diags := resp.Schema.ValidateImplementation(ctx); diags.HasError() {
		t.Fatalf("schema implementation is invalid: %v", diags.Errors())
	}
	return resp.Schema
}

// RoundTrip asserts that a model survives a write to state and a read back.
//
// This is what proves a resource's model matches its schema. The shared
// attribute blocks are embedded structs whose fields the framework promotes,
// so a mismatch between the schema half and the model half of a block shows up
// here rather than as a confusing failure during a real apply.
//
// Comparison happens on the Terraform values rather than on the Go structs:
// two equal framework values can hold different internal representations, for
// instance a collection built from a nil element slice versus an empty one.
func RoundTrip[T any](t *testing.T, res fwresource.Resource, value T) T {
	t.Helper()

	ctx := context.Background()
	resourceSchema := ResourceSchema(t, res)

	written := setState(t, ctx, resourceSchema, value)

	var readBack T
	if diags := written.Get(ctx, &readBack); diags.HasError() {
		t.Fatalf("reading model back from state: %v", diags.Errors())
	}

	rewritten := setState(t, ctx, resourceSchema, readBack)
	if !written.Raw.Equal(rewritten.Raw) {
		t.Errorf("model did not survive a state round-trip\nbefore: %s\nafter:  %s",
			written.Raw.String(), rewritten.Raw.String())
	}
	return readBack
}

func setState[T any](t *testing.T, ctx context.Context, resourceSchema schema.Schema, value T) tfsdk.State {
	t.Helper()

	// Set replaces Raw wholesale, so starting from a null object is fine.
	state := tfsdk.State{
		Raw:    tftypes.NewValue(resourceSchema.Type().TerraformType(ctx), nil),
		Schema: resourceSchema,
	}
	if diags := state.Set(ctx, value); diags.HasError() {
		t.Fatalf("setting state from model: %v", diags.Errors())
	}
	return state
}
