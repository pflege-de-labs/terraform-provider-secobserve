package tfutil_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/pflege-de-labs/terraform-provider-secobserve/internal/tfutil"
)

func emptyImportResponse(ctx context.Context) *resource.ImportStateResponse {
	schema := itemResourceSchema()
	return &resource.ImportStateResponse{State: tfsdk.State{
		Schema: schema,
		Raw:    tftypes.NewValue(schema.Type().TerraformType(ctx), nil),
	}}
}

func TestImportByIdentity_SkipsWhenIDSet(t *testing.T) {
	ctx := context.Background()
	resp := emptyImportResponse(ctx)

	if tfutil.ImportByIdentity(ctx, resource.ImportStateRequest{ID: "42"}, resp) {
		t.Fatal("handled a string import id")
	}
	if !resp.State.Raw.IsNull() || resp.Diagnostics.HasError() {
		t.Fatalf("state touched: %v %v", resp.State.Raw, resp.Diagnostics)
	}
}

func TestImportByIdentity_CopiesID(t *testing.T) {
	ctx := context.Background()
	identitySchema := idIdentitySchema().IdentitySchema
	req := resource.ImportStateRequest{Identity: &tfsdk.ResourceIdentity{
		Schema: identitySchema,
		Raw: tftypes.NewValue(identitySchema.Type().TerraformType(ctx), map[string]tftypes.Value{
			"id": tftypes.NewValue(tftypes.Number, 42),
		}),
	}}
	resp := emptyImportResponse(ctx)

	if !tfutil.ImportByIdentity(ctx, req, resp) {
		t.Fatal("identity import not handled")
	}
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	var id types.Int64
	resp.State.GetAttribute(ctx, path.Root("id"), &id)
	if id.ValueInt64() != 42 {
		t.Errorf("state id = %v, want 42", id)
	}
}

func TestSetIdentityID_NilIdentity(t *testing.T) {
	if diags := tfutil.SetIdentityID(context.Background(), nil, types.Int64Value(1)); diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
}
